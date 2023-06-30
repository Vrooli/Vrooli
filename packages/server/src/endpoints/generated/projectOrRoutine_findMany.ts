export const projectOrRoutine_findMany = {
  "fieldName": "projectOrRoutines",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "projectOrRoutines",
        "loc": {
          "start": 2481,
          "end": 2498
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 2499,
              "end": 2504
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 2507,
                "end": 2512
              }
            },
            "loc": {
              "start": 2506,
              "end": 2512
            }
          },
          "loc": {
            "start": 2499,
            "end": 2512
          }
        }
      ],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "edges",
              "loc": {
                "start": 2520,
                "end": 2525
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "cursor",
                    "loc": {
                      "start": 2536,
                      "end": 2542
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2536,
                    "end": 2542
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 2551,
                      "end": 2555
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Project",
                            "loc": {
                              "start": 2577,
                              "end": 2584
                            }
                          },
                          "loc": {
                            "start": 2577,
                            "end": 2584
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Project_list",
                                "loc": {
                                  "start": 2606,
                                  "end": 2618
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 2603,
                                "end": 2618
                              }
                            }
                          ],
                          "loc": {
                            "start": 2585,
                            "end": 2632
                          }
                        },
                        "loc": {
                          "start": 2570,
                          "end": 2632
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Routine",
                            "loc": {
                              "start": 2652,
                              "end": 2659
                            }
                          },
                          "loc": {
                            "start": 2652,
                            "end": 2659
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Routine_list",
                                "loc": {
                                  "start": 2681,
                                  "end": 2693
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 2678,
                                "end": 2693
                              }
                            }
                          ],
                          "loc": {
                            "start": 2660,
                            "end": 2707
                          }
                        },
                        "loc": {
                          "start": 2645,
                          "end": 2707
                        }
                      }
                    ],
                    "loc": {
                      "start": 2556,
                      "end": 2717
                    }
                  },
                  "loc": {
                    "start": 2551,
                    "end": 2717
                  }
                }
              ],
              "loc": {
                "start": 2526,
                "end": 2723
              }
            },
            "loc": {
              "start": 2520,
              "end": 2723
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 2728,
                "end": 2736
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasNextPage",
                    "loc": {
                      "start": 2747,
                      "end": 2758
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2747,
                    "end": 2758
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorProject",
                    "loc": {
                      "start": 2767,
                      "end": 2783
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2767,
                    "end": 2783
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorRoutine",
                    "loc": {
                      "start": 2792,
                      "end": 2808
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2792,
                    "end": 2808
                  }
                }
              ],
              "loc": {
                "start": 2737,
                "end": 2814
              }
            },
            "loc": {
              "start": 2728,
              "end": 2814
            }
          }
        ],
        "loc": {
          "start": 2514,
          "end": 2818
        }
      },
      "loc": {
        "start": 2481,
        "end": 2818
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 32,
          "end": 34
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 32,
        "end": 34
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 35,
          "end": 45
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 35,
        "end": 45
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 46,
          "end": 56
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 46,
        "end": 56
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 57,
          "end": 62
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 57,
        "end": 62
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 63,
          "end": 68
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 63,
        "end": 68
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 69,
          "end": 74
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 88,
                  "end": 100
                }
              },
              "loc": {
                "start": 88,
                "end": 100
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 114,
                      "end": 130
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 111,
                    "end": 130
                  }
                }
              ],
              "loc": {
                "start": 101,
                "end": 136
              }
            },
            "loc": {
              "start": 81,
              "end": 136
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 148,
                  "end": 152
                }
              },
              "loc": {
                "start": 148,
                "end": 152
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 166,
                      "end": 174
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 163,
                    "end": 174
                  }
                }
              ],
              "loc": {
                "start": 153,
                "end": 180
              }
            },
            "loc": {
              "start": 141,
              "end": 180
            }
          }
        ],
        "loc": {
          "start": 75,
          "end": 182
        }
      },
      "loc": {
        "start": 69,
        "end": 182
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 183,
          "end": 186
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 193,
                "end": 202
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 193,
              "end": 202
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 207,
                "end": 216
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 207,
              "end": 216
            }
          }
        ],
        "loc": {
          "start": 187,
          "end": 218
        }
      },
      "loc": {
        "start": 183,
        "end": 218
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 265,
          "end": 267
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 265,
        "end": 267
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 268,
          "end": 274
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 268,
        "end": 274
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 275,
          "end": 278
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canAddMembers",
              "loc": {
                "start": 285,
                "end": 298
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 285,
              "end": 298
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 303,
                "end": 312
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 303,
              "end": 312
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 317,
                "end": 328
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 317,
              "end": 328
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 333,
                "end": 342
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 333,
              "end": 342
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 347,
                "end": 356
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 347,
              "end": 356
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 361,
                "end": 368
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 361,
              "end": 368
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 373,
                "end": 385
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 373,
              "end": 385
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 390,
                "end": 398
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 390,
              "end": 398
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 403,
                "end": 417
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 428,
                      "end": 430
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 428,
                    "end": 430
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 439,
                      "end": 449
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 439,
                    "end": 449
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 458,
                      "end": 468
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 458,
                    "end": 468
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 477,
                      "end": 484
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 477,
                    "end": 484
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 493,
                      "end": 504
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 493,
                    "end": 504
                  }
                }
              ],
              "loc": {
                "start": 418,
                "end": 510
              }
            },
            "loc": {
              "start": 403,
              "end": 510
            }
          }
        ],
        "loc": {
          "start": 279,
          "end": 512
        }
      },
      "loc": {
        "start": 275,
        "end": 512
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 550,
          "end": 558
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 565,
                "end": 577
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 588,
                      "end": 590
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 588,
                    "end": 590
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 599,
                      "end": 607
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 599,
                    "end": 607
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 616,
                      "end": 627
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 616,
                    "end": 627
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 636,
                      "end": 640
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 636,
                    "end": 640
                  }
                }
              ],
              "loc": {
                "start": 578,
                "end": 646
              }
            },
            "loc": {
              "start": 565,
              "end": 646
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 651,
                "end": 653
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 651,
              "end": 653
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 658,
                "end": 668
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 658,
              "end": 668
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 673,
                "end": 683
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 673,
              "end": 683
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoriesCount",
              "loc": {
                "start": 688,
                "end": 704
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 688,
              "end": 704
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 709,
                "end": 717
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 709,
              "end": 717
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 722,
                "end": 731
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 722,
              "end": 731
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 736,
                "end": 748
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 736,
              "end": 748
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjectsCount",
              "loc": {
                "start": 753,
                "end": 769
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 753,
              "end": 769
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 774,
                "end": 784
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 774,
              "end": 784
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 789,
                "end": 801
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 789,
              "end": 801
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 806,
                "end": 818
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 806,
              "end": 818
            }
          }
        ],
        "loc": {
          "start": 559,
          "end": 820
        }
      },
      "loc": {
        "start": 550,
        "end": 820
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 821,
          "end": 823
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 821,
        "end": 823
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 824,
          "end": 834
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 824,
        "end": 834
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 835,
          "end": 845
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 835,
        "end": 845
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 846,
          "end": 855
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 846,
        "end": 855
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 856,
          "end": 867
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 856,
        "end": 867
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 868,
          "end": 874
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 884,
                "end": 894
              }
            },
            "directives": [],
            "loc": {
              "start": 881,
              "end": 894
            }
          }
        ],
        "loc": {
          "start": 875,
          "end": 896
        }
      },
      "loc": {
        "start": 868,
        "end": 896
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 897,
          "end": 902
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 916,
                  "end": 928
                }
              },
              "loc": {
                "start": 916,
                "end": 928
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 942,
                      "end": 958
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 939,
                    "end": 958
                  }
                }
              ],
              "loc": {
                "start": 929,
                "end": 964
              }
            },
            "loc": {
              "start": 909,
              "end": 964
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 976,
                  "end": 980
                }
              },
              "loc": {
                "start": 976,
                "end": 980
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 994,
                      "end": 1002
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 991,
                    "end": 1002
                  }
                }
              ],
              "loc": {
                "start": 981,
                "end": 1008
              }
            },
            "loc": {
              "start": 969,
              "end": 1008
            }
          }
        ],
        "loc": {
          "start": 903,
          "end": 1010
        }
      },
      "loc": {
        "start": 897,
        "end": 1010
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 1011,
          "end": 1022
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1011,
        "end": 1022
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 1023,
          "end": 1037
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1023,
        "end": 1037
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 1038,
          "end": 1043
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1038,
        "end": 1043
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 1044,
          "end": 1053
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1044,
        "end": 1053
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 1054,
          "end": 1058
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 1068,
                "end": 1076
              }
            },
            "directives": [],
            "loc": {
              "start": 1065,
              "end": 1076
            }
          }
        ],
        "loc": {
          "start": 1059,
          "end": 1078
        }
      },
      "loc": {
        "start": 1054,
        "end": 1078
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 1079,
          "end": 1093
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1079,
        "end": 1093
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 1094,
          "end": 1099
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1094,
        "end": 1099
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1100,
          "end": 1103
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 1110,
                "end": 1119
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1110,
              "end": 1119
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1124,
                "end": 1135
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1124,
              "end": 1135
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 1140,
                "end": 1151
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1140,
              "end": 1151
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1156,
                "end": 1165
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1156,
              "end": 1165
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1170,
                "end": 1177
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1170,
              "end": 1177
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 1182,
                "end": 1190
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1182,
              "end": 1190
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1195,
                "end": 1207
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1195,
              "end": 1207
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1212,
                "end": 1220
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1212,
              "end": 1220
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 1225,
                "end": 1233
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1225,
              "end": 1233
            }
          }
        ],
        "loc": {
          "start": 1104,
          "end": 1235
        }
      },
      "loc": {
        "start": 1100,
        "end": 1235
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 1273,
          "end": 1281
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1288,
                "end": 1300
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1311,
                      "end": 1313
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1311,
                    "end": 1313
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1322,
                      "end": 1330
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1322,
                    "end": 1330
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1339,
                      "end": 1350
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1339,
                    "end": 1350
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "instructions",
                    "loc": {
                      "start": 1359,
                      "end": 1371
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1359,
                    "end": 1371
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1380,
                      "end": 1384
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1380,
                    "end": 1384
                  }
                }
              ],
              "loc": {
                "start": 1301,
                "end": 1390
              }
            },
            "loc": {
              "start": 1288,
              "end": 1390
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1395,
                "end": 1397
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1395,
              "end": 1397
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1402,
                "end": 1412
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1402,
              "end": 1412
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1417,
                "end": 1427
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1417,
              "end": 1427
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 1432,
                "end": 1443
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1432,
              "end": 1443
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isAutomatable",
              "loc": {
                "start": 1448,
                "end": 1461
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1448,
              "end": 1461
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 1466,
                "end": 1476
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1466,
              "end": 1476
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 1481,
                "end": 1490
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1481,
              "end": 1490
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 1495,
                "end": 1503
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1495,
              "end": 1503
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1508,
                "end": 1517
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1508,
              "end": 1517
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 1522,
                "end": 1532
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1522,
              "end": 1532
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesStarted",
              "loc": {
                "start": 1537,
                "end": 1549
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1537,
              "end": 1549
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesCompleted",
              "loc": {
                "start": 1554,
                "end": 1568
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1554,
              "end": 1568
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "smartContractCallData",
              "loc": {
                "start": 1573,
                "end": 1594
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1573,
              "end": 1594
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "apiCallData",
              "loc": {
                "start": 1599,
                "end": 1610
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1599,
              "end": 1610
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 1615,
                "end": 1627
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1615,
              "end": 1627
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 1632,
                "end": 1644
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1632,
              "end": 1644
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 1649,
                "end": 1662
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1649,
              "end": 1662
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 1667,
                "end": 1689
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1667,
              "end": 1689
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 1694,
                "end": 1704
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1694,
              "end": 1704
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 1709,
                "end": 1720
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1709,
              "end": 1720
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodesCount",
              "loc": {
                "start": 1725,
                "end": 1735
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1725,
              "end": 1735
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodeLinksCount",
              "loc": {
                "start": 1740,
                "end": 1754
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1740,
              "end": 1754
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "outputsCount",
              "loc": {
                "start": 1759,
                "end": 1771
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1759,
              "end": 1771
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 1776,
                "end": 1788
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1776,
              "end": 1788
            }
          }
        ],
        "loc": {
          "start": 1282,
          "end": 1790
        }
      },
      "loc": {
        "start": 1273,
        "end": 1790
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1791,
          "end": 1793
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1791,
        "end": 1793
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1794,
          "end": 1804
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1794,
        "end": 1804
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1805,
          "end": 1815
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1805,
        "end": 1815
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 1816,
          "end": 1826
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1816,
        "end": 1826
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 1827,
          "end": 1836
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1827,
        "end": 1836
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 1837,
          "end": 1848
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1837,
        "end": 1848
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 1849,
          "end": 1855
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 1865,
                "end": 1875
              }
            },
            "directives": [],
            "loc": {
              "start": 1862,
              "end": 1875
            }
          }
        ],
        "loc": {
          "start": 1856,
          "end": 1877
        }
      },
      "loc": {
        "start": 1849,
        "end": 1877
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1878,
          "end": 1883
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 1897,
                  "end": 1909
                }
              },
              "loc": {
                "start": 1897,
                "end": 1909
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 1923,
                      "end": 1939
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1920,
                    "end": 1939
                  }
                }
              ],
              "loc": {
                "start": 1910,
                "end": 1945
              }
            },
            "loc": {
              "start": 1890,
              "end": 1945
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 1957,
                  "end": 1961
                }
              },
              "loc": {
                "start": 1957,
                "end": 1961
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 1975,
                      "end": 1983
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1972,
                    "end": 1983
                  }
                }
              ],
              "loc": {
                "start": 1962,
                "end": 1989
              }
            },
            "loc": {
              "start": 1950,
              "end": 1989
            }
          }
        ],
        "loc": {
          "start": 1884,
          "end": 1991
        }
      },
      "loc": {
        "start": 1878,
        "end": 1991
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 1992,
          "end": 2003
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1992,
        "end": 2003
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 2004,
          "end": 2018
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2004,
        "end": 2018
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 2019,
          "end": 2024
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2019,
        "end": 2024
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 2025,
          "end": 2034
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2025,
        "end": 2034
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 2035,
          "end": 2039
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 2049,
                "end": 2057
              }
            },
            "directives": [],
            "loc": {
              "start": 2046,
              "end": 2057
            }
          }
        ],
        "loc": {
          "start": 2040,
          "end": 2059
        }
      },
      "loc": {
        "start": 2035,
        "end": 2059
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 2060,
          "end": 2074
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2060,
        "end": 2074
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 2075,
          "end": 2080
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2075,
        "end": 2080
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2081,
          "end": 2084
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canComment",
              "loc": {
                "start": 2091,
                "end": 2101
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2091,
              "end": 2101
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 2106,
                "end": 2115
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2106,
              "end": 2115
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 2120,
                "end": 2131
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2120,
              "end": 2131
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2136,
                "end": 2145
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2136,
              "end": 2145
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 2150,
                "end": 2157
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2150,
              "end": 2157
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 2162,
                "end": 2170
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2162,
              "end": 2170
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2175,
                "end": 2187
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2175,
              "end": 2187
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 2192,
                "end": 2200
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2192,
              "end": 2200
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 2205,
                "end": 2213
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2205,
              "end": 2213
            }
          }
        ],
        "loc": {
          "start": 2085,
          "end": 2215
        }
      },
      "loc": {
        "start": 2081,
        "end": 2215
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2245,
          "end": 2247
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2245,
        "end": 2247
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 2248,
          "end": 2258
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2248,
        "end": 2258
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 2259,
          "end": 2262
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2259,
        "end": 2262
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 2263,
          "end": 2272
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2263,
        "end": 2272
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 2273,
          "end": 2285
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2292,
                "end": 2294
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2292,
              "end": 2294
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 2299,
                "end": 2307
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2299,
              "end": 2307
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 2312,
                "end": 2323
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2312,
              "end": 2323
            }
          }
        ],
        "loc": {
          "start": 2286,
          "end": 2325
        }
      },
      "loc": {
        "start": 2273,
        "end": 2325
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2326,
          "end": 2329
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOwn",
              "loc": {
                "start": 2336,
                "end": 2341
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2336,
              "end": 2341
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2346,
                "end": 2358
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2346,
              "end": 2358
            }
          }
        ],
        "loc": {
          "start": 2330,
          "end": 2360
        }
      },
      "loc": {
        "start": 2326,
        "end": 2360
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2391,
          "end": 2393
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2391,
        "end": 2393
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 2394,
          "end": 2399
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2394,
        "end": 2399
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 2400,
          "end": 2404
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2400,
        "end": 2404
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 2405,
          "end": 2411
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2405,
        "end": 2411
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 10,
          "end": 20
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 24,
            "end": 29
          }
        },
        "loc": {
          "start": 24,
          "end": 29
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 32,
                "end": 34
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 32,
              "end": 34
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 35,
                "end": 45
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 35,
              "end": 45
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 46,
                "end": 56
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 46,
              "end": 56
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 57,
                "end": 62
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 57,
              "end": 62
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 63,
                "end": 68
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 63,
              "end": 68
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 69,
                "end": 74
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 88,
                        "end": 100
                      }
                    },
                    "loc": {
                      "start": 88,
                      "end": 100
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 114,
                            "end": 130
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 111,
                          "end": 130
                        }
                      }
                    ],
                    "loc": {
                      "start": 101,
                      "end": 136
                    }
                  },
                  "loc": {
                    "start": 81,
                    "end": 136
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 148,
                        "end": 152
                      }
                    },
                    "loc": {
                      "start": 148,
                      "end": 152
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 166,
                            "end": 174
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 163,
                          "end": 174
                        }
                      }
                    ],
                    "loc": {
                      "start": 153,
                      "end": 180
                    }
                  },
                  "loc": {
                    "start": 141,
                    "end": 180
                  }
                }
              ],
              "loc": {
                "start": 75,
                "end": 182
              }
            },
            "loc": {
              "start": 69,
              "end": 182
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 183,
                "end": 186
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 193,
                      "end": 202
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 193,
                    "end": 202
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 207,
                      "end": 216
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 207,
                    "end": 216
                  }
                }
              ],
              "loc": {
                "start": 187,
                "end": 218
              }
            },
            "loc": {
              "start": 183,
              "end": 218
            }
          }
        ],
        "loc": {
          "start": 30,
          "end": 220
        }
      },
      "loc": {
        "start": 1,
        "end": 220
      }
    },
    "Organization_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_nav",
        "loc": {
          "start": 230,
          "end": 246
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 250,
            "end": 262
          }
        },
        "loc": {
          "start": 250,
          "end": 262
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 265,
                "end": 267
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 265,
              "end": 267
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 268,
                "end": 274
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 268,
              "end": 274
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 275,
                "end": 278
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 285,
                      "end": 298
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 285,
                    "end": 298
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 303,
                      "end": 312
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 303,
                    "end": 312
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 317,
                      "end": 328
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 317,
                    "end": 328
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 333,
                      "end": 342
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 333,
                    "end": 342
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 347,
                      "end": 356
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 347,
                    "end": 356
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 361,
                      "end": 368
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 361,
                    "end": 368
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 373,
                      "end": 385
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 373,
                    "end": 385
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 390,
                      "end": 398
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 390,
                    "end": 398
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 403,
                      "end": 417
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 428,
                            "end": 430
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 428,
                          "end": 430
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 439,
                            "end": 449
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 439,
                          "end": 449
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 458,
                            "end": 468
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 458,
                          "end": 468
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 477,
                            "end": 484
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 477,
                          "end": 484
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 493,
                            "end": 504
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 493,
                          "end": 504
                        }
                      }
                    ],
                    "loc": {
                      "start": 418,
                      "end": 510
                    }
                  },
                  "loc": {
                    "start": 403,
                    "end": 510
                  }
                }
              ],
              "loc": {
                "start": 279,
                "end": 512
              }
            },
            "loc": {
              "start": 275,
              "end": 512
            }
          }
        ],
        "loc": {
          "start": 263,
          "end": 514
        }
      },
      "loc": {
        "start": 221,
        "end": 514
      }
    },
    "Project_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_list",
        "loc": {
          "start": 524,
          "end": 536
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 540,
            "end": 547
          }
        },
        "loc": {
          "start": 540,
          "end": 547
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 550,
                "end": 558
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 565,
                      "end": 577
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 588,
                            "end": 590
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 588,
                          "end": 590
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 599,
                            "end": 607
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 599,
                          "end": 607
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 616,
                            "end": 627
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 616,
                          "end": 627
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 636,
                            "end": 640
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 636,
                          "end": 640
                        }
                      }
                    ],
                    "loc": {
                      "start": 578,
                      "end": 646
                    }
                  },
                  "loc": {
                    "start": 565,
                    "end": 646
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 651,
                      "end": 653
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 651,
                    "end": 653
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 658,
                      "end": 668
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 658,
                    "end": 668
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 673,
                      "end": 683
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 673,
                    "end": 683
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoriesCount",
                    "loc": {
                      "start": 688,
                      "end": 704
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 688,
                    "end": 704
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 709,
                      "end": 717
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 709,
                    "end": 717
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 722,
                      "end": 731
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 722,
                    "end": 731
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 736,
                      "end": 748
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 736,
                    "end": 748
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "runProjectsCount",
                    "loc": {
                      "start": 753,
                      "end": 769
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 753,
                    "end": 769
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 774,
                      "end": 784
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 774,
                    "end": 784
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 789,
                      "end": 801
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 789,
                    "end": 801
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 806,
                      "end": 818
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 806,
                    "end": 818
                  }
                }
              ],
              "loc": {
                "start": 559,
                "end": 820
              }
            },
            "loc": {
              "start": 550,
              "end": 820
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 821,
                "end": 823
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 821,
              "end": 823
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 824,
                "end": 834
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 824,
              "end": 834
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 835,
                "end": 845
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 835,
              "end": 845
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 846,
                "end": 855
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 846,
              "end": 855
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 856,
                "end": 867
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 856,
              "end": 867
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 868,
                "end": 874
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 884,
                      "end": 894
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 881,
                    "end": 894
                  }
                }
              ],
              "loc": {
                "start": 875,
                "end": 896
              }
            },
            "loc": {
              "start": 868,
              "end": 896
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 897,
                "end": 902
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 916,
                        "end": 928
                      }
                    },
                    "loc": {
                      "start": 916,
                      "end": 928
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 942,
                            "end": 958
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 939,
                          "end": 958
                        }
                      }
                    ],
                    "loc": {
                      "start": 929,
                      "end": 964
                    }
                  },
                  "loc": {
                    "start": 909,
                    "end": 964
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 976,
                        "end": 980
                      }
                    },
                    "loc": {
                      "start": 976,
                      "end": 980
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 994,
                            "end": 1002
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 991,
                          "end": 1002
                        }
                      }
                    ],
                    "loc": {
                      "start": 981,
                      "end": 1008
                    }
                  },
                  "loc": {
                    "start": 969,
                    "end": 1008
                  }
                }
              ],
              "loc": {
                "start": 903,
                "end": 1010
              }
            },
            "loc": {
              "start": 897,
              "end": 1010
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 1011,
                "end": 1022
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1011,
              "end": 1022
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 1023,
                "end": 1037
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1023,
              "end": 1037
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 1038,
                "end": 1043
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1038,
              "end": 1043
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1044,
                "end": 1053
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1044,
              "end": 1053
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 1054,
                "end": 1058
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 1068,
                      "end": 1076
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1065,
                    "end": 1076
                  }
                }
              ],
              "loc": {
                "start": 1059,
                "end": 1078
              }
            },
            "loc": {
              "start": 1054,
              "end": 1078
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 1079,
                "end": 1093
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1079,
              "end": 1093
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 1094,
                "end": 1099
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1094,
              "end": 1099
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1100,
                "end": 1103
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1110,
                      "end": 1119
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1110,
                    "end": 1119
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1124,
                      "end": 1135
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1124,
                    "end": 1135
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 1140,
                      "end": 1151
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1140,
                    "end": 1151
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1156,
                      "end": 1165
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1156,
                    "end": 1165
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1170,
                      "end": 1177
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1170,
                    "end": 1177
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 1182,
                      "end": 1190
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1182,
                    "end": 1190
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1195,
                      "end": 1207
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1195,
                    "end": 1207
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1212,
                      "end": 1220
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1212,
                    "end": 1220
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1225,
                      "end": 1233
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1225,
                    "end": 1233
                  }
                }
              ],
              "loc": {
                "start": 1104,
                "end": 1235
              }
            },
            "loc": {
              "start": 1100,
              "end": 1235
            }
          }
        ],
        "loc": {
          "start": 548,
          "end": 1237
        }
      },
      "loc": {
        "start": 515,
        "end": 1237
      }
    },
    "Routine_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_list",
        "loc": {
          "start": 1247,
          "end": 1259
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 1263,
            "end": 1270
          }
        },
        "loc": {
          "start": 1263,
          "end": 1270
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 1273,
                "end": 1281
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 1288,
                      "end": 1300
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1311,
                            "end": 1313
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1311,
                          "end": 1313
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 1322,
                            "end": 1330
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1322,
                          "end": 1330
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1339,
                            "end": 1350
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1339,
                          "end": 1350
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 1359,
                            "end": 1371
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1359,
                          "end": 1371
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1380,
                            "end": 1384
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1380,
                          "end": 1384
                        }
                      }
                    ],
                    "loc": {
                      "start": 1301,
                      "end": 1390
                    }
                  },
                  "loc": {
                    "start": 1288,
                    "end": 1390
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1395,
                      "end": 1397
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1395,
                    "end": 1397
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1402,
                      "end": 1412
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1402,
                    "end": 1412
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1417,
                      "end": 1427
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1417,
                    "end": 1427
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 1432,
                      "end": 1443
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1432,
                    "end": 1443
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 1448,
                      "end": 1461
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1448,
                    "end": 1461
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 1466,
                      "end": 1476
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1466,
                    "end": 1476
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 1481,
                      "end": 1490
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1481,
                    "end": 1490
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 1495,
                      "end": 1503
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1495,
                    "end": 1503
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1508,
                      "end": 1517
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1508,
                    "end": 1517
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 1522,
                      "end": 1532
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1522,
                    "end": 1532
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesStarted",
                    "loc": {
                      "start": 1537,
                      "end": 1549
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1537,
                    "end": 1549
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesCompleted",
                    "loc": {
                      "start": 1554,
                      "end": 1568
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1554,
                    "end": 1568
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "smartContractCallData",
                    "loc": {
                      "start": 1573,
                      "end": 1594
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1573,
                    "end": 1594
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apiCallData",
                    "loc": {
                      "start": 1599,
                      "end": 1610
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1599,
                    "end": 1610
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 1615,
                      "end": 1627
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1615,
                    "end": 1627
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 1632,
                      "end": 1644
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1632,
                    "end": 1644
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 1649,
                      "end": 1662
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1649,
                    "end": 1662
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 1667,
                      "end": 1689
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1667,
                    "end": 1689
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 1694,
                      "end": 1704
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1694,
                    "end": 1704
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 1709,
                      "end": 1720
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1709,
                    "end": 1720
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodesCount",
                    "loc": {
                      "start": 1725,
                      "end": 1735
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1725,
                    "end": 1735
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodeLinksCount",
                    "loc": {
                      "start": 1740,
                      "end": 1754
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1740,
                    "end": 1754
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "outputsCount",
                    "loc": {
                      "start": 1759,
                      "end": 1771
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1759,
                    "end": 1771
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 1776,
                      "end": 1788
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1776,
                    "end": 1788
                  }
                }
              ],
              "loc": {
                "start": 1282,
                "end": 1790
              }
            },
            "loc": {
              "start": 1273,
              "end": 1790
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1791,
                "end": 1793
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1791,
              "end": 1793
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1794,
                "end": 1804
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1794,
              "end": 1804
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1805,
                "end": 1815
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1805,
              "end": 1815
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 1816,
                "end": 1826
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1816,
              "end": 1826
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1827,
                "end": 1836
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1827,
              "end": 1836
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 1837,
                "end": 1848
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1837,
              "end": 1848
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 1849,
                "end": 1855
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 1865,
                      "end": 1875
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1862,
                    "end": 1875
                  }
                }
              ],
              "loc": {
                "start": 1856,
                "end": 1877
              }
            },
            "loc": {
              "start": 1849,
              "end": 1877
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1878,
                "end": 1883
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 1897,
                        "end": 1909
                      }
                    },
                    "loc": {
                      "start": 1897,
                      "end": 1909
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 1923,
                            "end": 1939
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1920,
                          "end": 1939
                        }
                      }
                    ],
                    "loc": {
                      "start": 1910,
                      "end": 1945
                    }
                  },
                  "loc": {
                    "start": 1890,
                    "end": 1945
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 1957,
                        "end": 1961
                      }
                    },
                    "loc": {
                      "start": 1957,
                      "end": 1961
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 1975,
                            "end": 1983
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1972,
                          "end": 1983
                        }
                      }
                    ],
                    "loc": {
                      "start": 1962,
                      "end": 1989
                    }
                  },
                  "loc": {
                    "start": 1950,
                    "end": 1989
                  }
                }
              ],
              "loc": {
                "start": 1884,
                "end": 1991
              }
            },
            "loc": {
              "start": 1878,
              "end": 1991
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 1992,
                "end": 2003
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1992,
              "end": 2003
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 2004,
                "end": 2018
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2004,
              "end": 2018
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 2019,
                "end": 2024
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2019,
              "end": 2024
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2025,
                "end": 2034
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2025,
              "end": 2034
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 2035,
                "end": 2039
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 2049,
                      "end": 2057
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2046,
                    "end": 2057
                  }
                }
              ],
              "loc": {
                "start": 2040,
                "end": 2059
              }
            },
            "loc": {
              "start": 2035,
              "end": 2059
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 2060,
                "end": 2074
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2060,
              "end": 2074
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 2075,
                "end": 2080
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2075,
              "end": 2080
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2081,
                "end": 2084
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 2091,
                      "end": 2101
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2091,
                    "end": 2101
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2106,
                      "end": 2115
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2106,
                    "end": 2115
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2120,
                      "end": 2131
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2120,
                    "end": 2131
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2136,
                      "end": 2145
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2136,
                    "end": 2145
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2150,
                      "end": 2157
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2150,
                    "end": 2157
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 2162,
                      "end": 2170
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2162,
                    "end": 2170
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2175,
                      "end": 2187
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2175,
                    "end": 2187
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2192,
                      "end": 2200
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2192,
                    "end": 2200
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 2205,
                      "end": 2213
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2205,
                    "end": 2213
                  }
                }
              ],
              "loc": {
                "start": 2085,
                "end": 2215
              }
            },
            "loc": {
              "start": 2081,
              "end": 2215
            }
          }
        ],
        "loc": {
          "start": 1271,
          "end": 2217
        }
      },
      "loc": {
        "start": 1238,
        "end": 2217
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 2227,
          "end": 2235
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 2239,
            "end": 2242
          }
        },
        "loc": {
          "start": 2239,
          "end": 2242
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2245,
                "end": 2247
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2245,
              "end": 2247
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2248,
                "end": 2258
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2248,
              "end": 2258
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 2259,
                "end": 2262
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2259,
              "end": 2262
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2263,
                "end": 2272
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2263,
              "end": 2272
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 2273,
                "end": 2285
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2292,
                      "end": 2294
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2292,
                    "end": 2294
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2299,
                      "end": 2307
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2299,
                    "end": 2307
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2312,
                      "end": 2323
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2312,
                    "end": 2323
                  }
                }
              ],
              "loc": {
                "start": 2286,
                "end": 2325
              }
            },
            "loc": {
              "start": 2273,
              "end": 2325
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2326,
                "end": 2329
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isOwn",
                    "loc": {
                      "start": 2336,
                      "end": 2341
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2336,
                    "end": 2341
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2346,
                      "end": 2358
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2346,
                    "end": 2358
                  }
                }
              ],
              "loc": {
                "start": 2330,
                "end": 2360
              }
            },
            "loc": {
              "start": 2326,
              "end": 2360
            }
          }
        ],
        "loc": {
          "start": 2243,
          "end": 2362
        }
      },
      "loc": {
        "start": 2218,
        "end": 2362
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 2372,
          "end": 2380
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 2384,
            "end": 2388
          }
        },
        "loc": {
          "start": 2384,
          "end": 2388
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2391,
                "end": 2393
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2391,
              "end": 2393
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 2394,
                "end": 2399
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2394,
              "end": 2399
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 2400,
                "end": 2404
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2400,
              "end": 2404
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2405,
                "end": 2411
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2405,
              "end": 2411
            }
          }
        ],
        "loc": {
          "start": 2389,
          "end": 2413
        }
      },
      "loc": {
        "start": 2363,
        "end": 2413
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "projectOrRoutines",
      "loc": {
        "start": 2421,
        "end": 2438
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 2440,
              "end": 2445
            }
          },
          "loc": {
            "start": 2439,
            "end": 2445
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "ProjectOrRoutineSearchInput",
              "loc": {
                "start": 2447,
                "end": 2474
              }
            },
            "loc": {
              "start": 2447,
              "end": 2474
            }
          },
          "loc": {
            "start": 2447,
            "end": 2475
          }
        },
        "directives": [],
        "loc": {
          "start": 2439,
          "end": 2475
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "projectOrRoutines",
            "loc": {
              "start": 2481,
              "end": 2498
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 2499,
                  "end": 2504
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 2507,
                    "end": 2512
                  }
                },
                "loc": {
                  "start": 2506,
                  "end": 2512
                }
              },
              "loc": {
                "start": 2499,
                "end": 2512
              }
            }
          ],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "edges",
                  "loc": {
                    "start": 2520,
                    "end": 2525
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "cursor",
                        "loc": {
                          "start": 2536,
                          "end": 2542
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2536,
                        "end": 2542
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 2551,
                          "end": 2555
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Project",
                                "loc": {
                                  "start": 2577,
                                  "end": 2584
                                }
                              },
                              "loc": {
                                "start": 2577,
                                "end": 2584
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Project_list",
                                    "loc": {
                                      "start": 2606,
                                      "end": 2618
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 2603,
                                    "end": 2618
                                  }
                                }
                              ],
                              "loc": {
                                "start": 2585,
                                "end": 2632
                              }
                            },
                            "loc": {
                              "start": 2570,
                              "end": 2632
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Routine",
                                "loc": {
                                  "start": 2652,
                                  "end": 2659
                                }
                              },
                              "loc": {
                                "start": 2652,
                                "end": 2659
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Routine_list",
                                    "loc": {
                                      "start": 2681,
                                      "end": 2693
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 2678,
                                    "end": 2693
                                  }
                                }
                              ],
                              "loc": {
                                "start": 2660,
                                "end": 2707
                              }
                            },
                            "loc": {
                              "start": 2645,
                              "end": 2707
                            }
                          }
                        ],
                        "loc": {
                          "start": 2556,
                          "end": 2717
                        }
                      },
                      "loc": {
                        "start": 2551,
                        "end": 2717
                      }
                    }
                  ],
                  "loc": {
                    "start": 2526,
                    "end": 2723
                  }
                },
                "loc": {
                  "start": 2520,
                  "end": 2723
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 2728,
                    "end": 2736
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "hasNextPage",
                        "loc": {
                          "start": 2747,
                          "end": 2758
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2747,
                        "end": 2758
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorProject",
                        "loc": {
                          "start": 2767,
                          "end": 2783
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2767,
                        "end": 2783
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorRoutine",
                        "loc": {
                          "start": 2792,
                          "end": 2808
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2792,
                        "end": 2808
                      }
                    }
                  ],
                  "loc": {
                    "start": 2737,
                    "end": 2814
                  }
                },
                "loc": {
                  "start": 2728,
                  "end": 2814
                }
              }
            ],
            "loc": {
              "start": 2514,
              "end": 2818
            }
          },
          "loc": {
            "start": 2481,
            "end": 2818
          }
        }
      ],
      "loc": {
        "start": 2477,
        "end": 2820
      }
    },
    "loc": {
      "start": 2415,
      "end": 2820
    }
  },
  "variableValues": {},
  "path": {
    "key": "projectOrRoutine_findMany"
  }
} as const;
