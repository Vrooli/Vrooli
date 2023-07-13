export const projectOrRoutine_findMany = {
  "fieldName": "projectOrRoutines",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "projectOrRoutines",
        "loc": {
          "start": 2531,
          "end": 2548
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 2549,
              "end": 2554
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 2557,
                "end": 2562
              }
            },
            "loc": {
              "start": 2556,
              "end": 2562
            }
          },
          "loc": {
            "start": 2549,
            "end": 2562
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
                "start": 2570,
                "end": 2575
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
                      "start": 2586,
                      "end": 2592
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2586,
                    "end": 2592
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 2601,
                      "end": 2605
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
                              "start": 2627,
                              "end": 2634
                            }
                          },
                          "loc": {
                            "start": 2627,
                            "end": 2634
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
                                  "start": 2656,
                                  "end": 2668
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 2653,
                                "end": 2668
                              }
                            }
                          ],
                          "loc": {
                            "start": 2635,
                            "end": 2682
                          }
                        },
                        "loc": {
                          "start": 2620,
                          "end": 2682
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
                              "start": 2702,
                              "end": 2709
                            }
                          },
                          "loc": {
                            "start": 2702,
                            "end": 2709
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
                                  "start": 2731,
                                  "end": 2743
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 2728,
                                "end": 2743
                              }
                            }
                          ],
                          "loc": {
                            "start": 2710,
                            "end": 2757
                          }
                        },
                        "loc": {
                          "start": 2695,
                          "end": 2757
                        }
                      }
                    ],
                    "loc": {
                      "start": 2606,
                      "end": 2767
                    }
                  },
                  "loc": {
                    "start": 2601,
                    "end": 2767
                  }
                }
              ],
              "loc": {
                "start": 2576,
                "end": 2773
              }
            },
            "loc": {
              "start": 2570,
              "end": 2773
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 2778,
                "end": 2786
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
                      "start": 2797,
                      "end": 2808
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2797,
                    "end": 2808
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorProject",
                    "loc": {
                      "start": 2817,
                      "end": 2833
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2817,
                    "end": 2833
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorRoutine",
                    "loc": {
                      "start": 2842,
                      "end": 2858
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2842,
                    "end": 2858
                  }
                }
              ],
              "loc": {
                "start": 2787,
                "end": 2864
              }
            },
            "loc": {
              "start": 2778,
              "end": 2864
            }
          }
        ],
        "loc": {
          "start": 2564,
          "end": 2868
        }
      },
      "loc": {
        "start": 2531,
        "end": 2868
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
        "value": "bannerImage",
        "loc": {
          "start": 268,
          "end": 279
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 268,
        "end": 279
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 280,
          "end": 286
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 280,
        "end": 286
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 287,
          "end": 299
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 287,
        "end": 299
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 300,
          "end": 303
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
                "start": 310,
                "end": 323
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 310,
              "end": 323
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 328,
                "end": 337
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 328,
              "end": 337
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 342,
                "end": 353
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 342,
              "end": 353
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 358,
                "end": 367
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 358,
              "end": 367
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 372,
                "end": 381
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 372,
              "end": 381
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 386,
                "end": 393
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 386,
              "end": 393
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 398,
                "end": 410
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 398,
              "end": 410
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 415,
                "end": 423
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 415,
              "end": 423
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 428,
                "end": 442
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
                      "start": 453,
                      "end": 455
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 453,
                    "end": 455
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 464,
                      "end": 474
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 464,
                    "end": 474
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 483,
                      "end": 493
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 483,
                    "end": 493
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 502,
                      "end": 509
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 502,
                    "end": 509
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 518,
                      "end": 529
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 518,
                    "end": 529
                  }
                }
              ],
              "loc": {
                "start": 443,
                "end": 535
              }
            },
            "loc": {
              "start": 428,
              "end": 535
            }
          }
        ],
        "loc": {
          "start": 304,
          "end": 537
        }
      },
      "loc": {
        "start": 300,
        "end": 537
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 575,
          "end": 583
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
                "start": 590,
                "end": 602
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
                      "start": 613,
                      "end": 615
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 613,
                    "end": 615
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 624,
                      "end": 632
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 624,
                    "end": 632
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 641,
                      "end": 652
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 641,
                    "end": 652
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 661,
                      "end": 665
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 661,
                    "end": 665
                  }
                }
              ],
              "loc": {
                "start": 603,
                "end": 671
              }
            },
            "loc": {
              "start": 590,
              "end": 671
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 676,
                "end": 678
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 676,
              "end": 678
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 683,
                "end": 693
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 683,
              "end": 693
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 698,
                "end": 708
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 698,
              "end": 708
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoriesCount",
              "loc": {
                "start": 713,
                "end": 729
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 713,
              "end": 729
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 734,
                "end": 742
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 734,
              "end": 742
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 747,
                "end": 756
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 747,
              "end": 756
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 761,
                "end": 773
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 761,
              "end": 773
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjectsCount",
              "loc": {
                "start": 778,
                "end": 794
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 778,
              "end": 794
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 799,
                "end": 809
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 799,
              "end": 809
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 814,
                "end": 826
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 814,
              "end": 826
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 831,
                "end": 843
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 831,
              "end": 843
            }
          }
        ],
        "loc": {
          "start": 584,
          "end": 845
        }
      },
      "loc": {
        "start": 575,
        "end": 845
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 846,
          "end": 848
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 846,
        "end": 848
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 849,
          "end": 859
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 849,
        "end": 859
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 860,
          "end": 870
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 860,
        "end": 870
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 871,
          "end": 880
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 871,
        "end": 880
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 881,
          "end": 892
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 881,
        "end": 892
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 893,
          "end": 899
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
                "start": 909,
                "end": 919
              }
            },
            "directives": [],
            "loc": {
              "start": 906,
              "end": 919
            }
          }
        ],
        "loc": {
          "start": 900,
          "end": 921
        }
      },
      "loc": {
        "start": 893,
        "end": 921
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 922,
          "end": 927
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
                  "start": 941,
                  "end": 953
                }
              },
              "loc": {
                "start": 941,
                "end": 953
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
                      "start": 967,
                      "end": 983
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 964,
                    "end": 983
                  }
                }
              ],
              "loc": {
                "start": 954,
                "end": 989
              }
            },
            "loc": {
              "start": 934,
              "end": 989
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
                  "start": 1001,
                  "end": 1005
                }
              },
              "loc": {
                "start": 1001,
                "end": 1005
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
                      "start": 1019,
                      "end": 1027
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1016,
                    "end": 1027
                  }
                }
              ],
              "loc": {
                "start": 1006,
                "end": 1033
              }
            },
            "loc": {
              "start": 994,
              "end": 1033
            }
          }
        ],
        "loc": {
          "start": 928,
          "end": 1035
        }
      },
      "loc": {
        "start": 922,
        "end": 1035
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 1036,
          "end": 1047
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1036,
        "end": 1047
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 1048,
          "end": 1062
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1048,
        "end": 1062
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 1063,
          "end": 1068
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1063,
        "end": 1068
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 1069,
          "end": 1078
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1069,
        "end": 1078
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 1079,
          "end": 1083
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
                "start": 1093,
                "end": 1101
              }
            },
            "directives": [],
            "loc": {
              "start": 1090,
              "end": 1101
            }
          }
        ],
        "loc": {
          "start": 1084,
          "end": 1103
        }
      },
      "loc": {
        "start": 1079,
        "end": 1103
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 1104,
          "end": 1118
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1104,
        "end": 1118
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 1119,
          "end": 1124
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1119,
        "end": 1124
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1125,
          "end": 1128
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
                "start": 1135,
                "end": 1144
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1135,
              "end": 1144
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1149,
                "end": 1160
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1149,
              "end": 1160
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 1165,
                "end": 1176
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1165,
              "end": 1176
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1181,
                "end": 1190
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1181,
              "end": 1190
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1195,
                "end": 1202
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1195,
              "end": 1202
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 1207,
                "end": 1215
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1207,
              "end": 1215
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1220,
                "end": 1232
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1220,
              "end": 1232
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1237,
                "end": 1245
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1237,
              "end": 1245
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 1250,
                "end": 1258
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1250,
              "end": 1258
            }
          }
        ],
        "loc": {
          "start": 1129,
          "end": 1260
        }
      },
      "loc": {
        "start": 1125,
        "end": 1260
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 1298,
          "end": 1306
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
                "start": 1313,
                "end": 1325
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
                      "start": 1336,
                      "end": 1338
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1336,
                    "end": 1338
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1347,
                      "end": 1355
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1347,
                    "end": 1355
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1364,
                      "end": 1375
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1364,
                    "end": 1375
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "instructions",
                    "loc": {
                      "start": 1384,
                      "end": 1396
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1384,
                    "end": 1396
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1405,
                      "end": 1409
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1405,
                    "end": 1409
                  }
                }
              ],
              "loc": {
                "start": 1326,
                "end": 1415
              }
            },
            "loc": {
              "start": 1313,
              "end": 1415
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1420,
                "end": 1422
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1420,
              "end": 1422
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1427,
                "end": 1437
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1427,
              "end": 1437
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1442,
                "end": 1452
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1442,
              "end": 1452
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 1457,
                "end": 1468
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1457,
              "end": 1468
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isAutomatable",
              "loc": {
                "start": 1473,
                "end": 1486
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1473,
              "end": 1486
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 1491,
                "end": 1501
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1491,
              "end": 1501
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 1506,
                "end": 1515
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1506,
              "end": 1515
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 1520,
                "end": 1528
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1520,
              "end": 1528
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1533,
                "end": 1542
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1533,
              "end": 1542
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 1547,
                "end": 1557
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1547,
              "end": 1557
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesStarted",
              "loc": {
                "start": 1562,
                "end": 1574
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1562,
              "end": 1574
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesCompleted",
              "loc": {
                "start": 1579,
                "end": 1593
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1579,
              "end": 1593
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "smartContractCallData",
              "loc": {
                "start": 1598,
                "end": 1619
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1598,
              "end": 1619
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "apiCallData",
              "loc": {
                "start": 1624,
                "end": 1635
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1624,
              "end": 1635
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 1640,
                "end": 1652
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1640,
              "end": 1652
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 1657,
                "end": 1669
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1657,
              "end": 1669
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 1674,
                "end": 1687
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1674,
              "end": 1687
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 1692,
                "end": 1714
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1692,
              "end": 1714
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 1719,
                "end": 1729
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1719,
              "end": 1729
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 1734,
                "end": 1745
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1734,
              "end": 1745
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodesCount",
              "loc": {
                "start": 1750,
                "end": 1760
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1750,
              "end": 1760
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodeLinksCount",
              "loc": {
                "start": 1765,
                "end": 1779
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1765,
              "end": 1779
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "outputsCount",
              "loc": {
                "start": 1784,
                "end": 1796
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1784,
              "end": 1796
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 1801,
                "end": 1813
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1801,
              "end": 1813
            }
          }
        ],
        "loc": {
          "start": 1307,
          "end": 1815
        }
      },
      "loc": {
        "start": 1298,
        "end": 1815
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1816,
          "end": 1818
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1816,
        "end": 1818
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1819,
          "end": 1829
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1819,
        "end": 1829
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1830,
          "end": 1840
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1830,
        "end": 1840
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 1841,
          "end": 1851
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1841,
        "end": 1851
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 1852,
          "end": 1861
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1852,
        "end": 1861
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 1862,
          "end": 1873
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1862,
        "end": 1873
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 1874,
          "end": 1880
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
                "start": 1890,
                "end": 1900
              }
            },
            "directives": [],
            "loc": {
              "start": 1887,
              "end": 1900
            }
          }
        ],
        "loc": {
          "start": 1881,
          "end": 1902
        }
      },
      "loc": {
        "start": 1874,
        "end": 1902
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1903,
          "end": 1908
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
                  "start": 1922,
                  "end": 1934
                }
              },
              "loc": {
                "start": 1922,
                "end": 1934
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
                      "start": 1948,
                      "end": 1964
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1945,
                    "end": 1964
                  }
                }
              ],
              "loc": {
                "start": 1935,
                "end": 1970
              }
            },
            "loc": {
              "start": 1915,
              "end": 1970
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
                  "start": 1982,
                  "end": 1986
                }
              },
              "loc": {
                "start": 1982,
                "end": 1986
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
                      "start": 2000,
                      "end": 2008
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1997,
                    "end": 2008
                  }
                }
              ],
              "loc": {
                "start": 1987,
                "end": 2014
              }
            },
            "loc": {
              "start": 1975,
              "end": 2014
            }
          }
        ],
        "loc": {
          "start": 1909,
          "end": 2016
        }
      },
      "loc": {
        "start": 1903,
        "end": 2016
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 2017,
          "end": 2028
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2017,
        "end": 2028
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 2029,
          "end": 2043
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2029,
        "end": 2043
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 2044,
          "end": 2049
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2044,
        "end": 2049
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 2050,
          "end": 2059
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2050,
        "end": 2059
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 2060,
          "end": 2064
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
                "start": 2074,
                "end": 2082
              }
            },
            "directives": [],
            "loc": {
              "start": 2071,
              "end": 2082
            }
          }
        ],
        "loc": {
          "start": 2065,
          "end": 2084
        }
      },
      "loc": {
        "start": 2060,
        "end": 2084
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 2085,
          "end": 2099
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2085,
        "end": 2099
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 2100,
          "end": 2105
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2100,
        "end": 2105
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2106,
          "end": 2109
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
                "start": 2116,
                "end": 2126
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2116,
              "end": 2126
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 2131,
                "end": 2140
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2131,
              "end": 2140
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 2145,
                "end": 2156
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2145,
              "end": 2156
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2161,
                "end": 2170
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2161,
              "end": 2170
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 2175,
                "end": 2182
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2175,
              "end": 2182
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 2187,
                "end": 2195
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2187,
              "end": 2195
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2200,
                "end": 2212
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2200,
              "end": 2212
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 2217,
                "end": 2225
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2217,
              "end": 2225
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 2230,
                "end": 2238
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2230,
              "end": 2238
            }
          }
        ],
        "loc": {
          "start": 2110,
          "end": 2240
        }
      },
      "loc": {
        "start": 2106,
        "end": 2240
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2270,
          "end": 2272
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2270,
        "end": 2272
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 2273,
          "end": 2283
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2273,
        "end": 2283
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 2284,
          "end": 2287
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2284,
        "end": 2287
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 2288,
          "end": 2297
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2288,
        "end": 2297
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 2298,
          "end": 2310
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
                "start": 2317,
                "end": 2319
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2317,
              "end": 2319
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 2324,
                "end": 2332
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2324,
              "end": 2332
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 2337,
                "end": 2348
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2337,
              "end": 2348
            }
          }
        ],
        "loc": {
          "start": 2311,
          "end": 2350
        }
      },
      "loc": {
        "start": 2298,
        "end": 2350
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2351,
          "end": 2354
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
                "start": 2361,
                "end": 2366
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2361,
              "end": 2366
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2371,
                "end": 2383
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2371,
              "end": 2383
            }
          }
        ],
        "loc": {
          "start": 2355,
          "end": 2385
        }
      },
      "loc": {
        "start": 2351,
        "end": 2385
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2416,
          "end": 2418
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2416,
        "end": 2418
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 2419,
          "end": 2430
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2419,
        "end": 2430
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 2431,
          "end": 2437
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2431,
        "end": 2437
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 2438,
          "end": 2443
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2438,
        "end": 2443
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 2444,
          "end": 2448
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2444,
        "end": 2448
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 2449,
          "end": 2461
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2449,
        "end": 2461
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
              "value": "bannerImage",
              "loc": {
                "start": 268,
                "end": 279
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 268,
              "end": 279
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 280,
                "end": 286
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 280,
              "end": 286
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 287,
                "end": 299
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 287,
              "end": 299
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 300,
                "end": 303
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
                      "start": 310,
                      "end": 323
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 310,
                    "end": 323
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 328,
                      "end": 337
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 328,
                    "end": 337
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 342,
                      "end": 353
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 342,
                    "end": 353
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 358,
                      "end": 367
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 358,
                    "end": 367
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 372,
                      "end": 381
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 372,
                    "end": 381
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 386,
                      "end": 393
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 386,
                    "end": 393
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 398,
                      "end": 410
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 398,
                    "end": 410
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 415,
                      "end": 423
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 415,
                    "end": 423
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 428,
                      "end": 442
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
                            "start": 453,
                            "end": 455
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 453,
                          "end": 455
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 464,
                            "end": 474
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 464,
                          "end": 474
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 483,
                            "end": 493
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 483,
                          "end": 493
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 502,
                            "end": 509
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 502,
                          "end": 509
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 518,
                            "end": 529
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 518,
                          "end": 529
                        }
                      }
                    ],
                    "loc": {
                      "start": 443,
                      "end": 535
                    }
                  },
                  "loc": {
                    "start": 428,
                    "end": 535
                  }
                }
              ],
              "loc": {
                "start": 304,
                "end": 537
              }
            },
            "loc": {
              "start": 300,
              "end": 537
            }
          }
        ],
        "loc": {
          "start": 263,
          "end": 539
        }
      },
      "loc": {
        "start": 221,
        "end": 539
      }
    },
    "Project_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_list",
        "loc": {
          "start": 549,
          "end": 561
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 565,
            "end": 572
          }
        },
        "loc": {
          "start": 565,
          "end": 572
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
                "start": 575,
                "end": 583
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
                      "start": 590,
                      "end": 602
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
                            "start": 613,
                            "end": 615
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 613,
                          "end": 615
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 624,
                            "end": 632
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 624,
                          "end": 632
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 641,
                            "end": 652
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 641,
                          "end": 652
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 661,
                            "end": 665
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 661,
                          "end": 665
                        }
                      }
                    ],
                    "loc": {
                      "start": 603,
                      "end": 671
                    }
                  },
                  "loc": {
                    "start": 590,
                    "end": 671
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 676,
                      "end": 678
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 676,
                    "end": 678
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 683,
                      "end": 693
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 683,
                    "end": 693
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 698,
                      "end": 708
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 698,
                    "end": 708
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoriesCount",
                    "loc": {
                      "start": 713,
                      "end": 729
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 713,
                    "end": 729
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 734,
                      "end": 742
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 734,
                    "end": 742
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 747,
                      "end": 756
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 747,
                    "end": 756
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 761,
                      "end": 773
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 761,
                    "end": 773
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "runProjectsCount",
                    "loc": {
                      "start": 778,
                      "end": 794
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 778,
                    "end": 794
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 799,
                      "end": 809
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 799,
                    "end": 809
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 814,
                      "end": 826
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 814,
                    "end": 826
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 831,
                      "end": 843
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 831,
                    "end": 843
                  }
                }
              ],
              "loc": {
                "start": 584,
                "end": 845
              }
            },
            "loc": {
              "start": 575,
              "end": 845
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 846,
                "end": 848
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 846,
              "end": 848
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 849,
                "end": 859
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 849,
              "end": 859
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 860,
                "end": 870
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 860,
              "end": 870
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 871,
                "end": 880
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 871,
              "end": 880
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 881,
                "end": 892
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 881,
              "end": 892
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 893,
                "end": 899
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
                      "start": 909,
                      "end": 919
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 906,
                    "end": 919
                  }
                }
              ],
              "loc": {
                "start": 900,
                "end": 921
              }
            },
            "loc": {
              "start": 893,
              "end": 921
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 922,
                "end": 927
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
                        "start": 941,
                        "end": 953
                      }
                    },
                    "loc": {
                      "start": 941,
                      "end": 953
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
                            "start": 967,
                            "end": 983
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 964,
                          "end": 983
                        }
                      }
                    ],
                    "loc": {
                      "start": 954,
                      "end": 989
                    }
                  },
                  "loc": {
                    "start": 934,
                    "end": 989
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
                        "start": 1001,
                        "end": 1005
                      }
                    },
                    "loc": {
                      "start": 1001,
                      "end": 1005
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
                            "start": 1019,
                            "end": 1027
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1016,
                          "end": 1027
                        }
                      }
                    ],
                    "loc": {
                      "start": 1006,
                      "end": 1033
                    }
                  },
                  "loc": {
                    "start": 994,
                    "end": 1033
                  }
                }
              ],
              "loc": {
                "start": 928,
                "end": 1035
              }
            },
            "loc": {
              "start": 922,
              "end": 1035
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 1036,
                "end": 1047
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1036,
              "end": 1047
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 1048,
                "end": 1062
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1048,
              "end": 1062
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 1063,
                "end": 1068
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1063,
              "end": 1068
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1069,
                "end": 1078
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1069,
              "end": 1078
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 1079,
                "end": 1083
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
                      "start": 1093,
                      "end": 1101
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1090,
                    "end": 1101
                  }
                }
              ],
              "loc": {
                "start": 1084,
                "end": 1103
              }
            },
            "loc": {
              "start": 1079,
              "end": 1103
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 1104,
                "end": 1118
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1104,
              "end": 1118
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 1119,
                "end": 1124
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1119,
              "end": 1124
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1125,
                "end": 1128
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
                      "start": 1135,
                      "end": 1144
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1135,
                    "end": 1144
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1149,
                      "end": 1160
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1149,
                    "end": 1160
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 1165,
                      "end": 1176
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1165,
                    "end": 1176
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1181,
                      "end": 1190
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1181,
                    "end": 1190
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1195,
                      "end": 1202
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1195,
                    "end": 1202
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 1207,
                      "end": 1215
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1207,
                    "end": 1215
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1220,
                      "end": 1232
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1220,
                    "end": 1232
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1237,
                      "end": 1245
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1237,
                    "end": 1245
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1250,
                      "end": 1258
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1250,
                    "end": 1258
                  }
                }
              ],
              "loc": {
                "start": 1129,
                "end": 1260
              }
            },
            "loc": {
              "start": 1125,
              "end": 1260
            }
          }
        ],
        "loc": {
          "start": 573,
          "end": 1262
        }
      },
      "loc": {
        "start": 540,
        "end": 1262
      }
    },
    "Routine_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_list",
        "loc": {
          "start": 1272,
          "end": 1284
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 1288,
            "end": 1295
          }
        },
        "loc": {
          "start": 1288,
          "end": 1295
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
                "start": 1298,
                "end": 1306
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
                      "start": 1313,
                      "end": 1325
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
                            "start": 1336,
                            "end": 1338
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1336,
                          "end": 1338
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 1347,
                            "end": 1355
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1347,
                          "end": 1355
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1364,
                            "end": 1375
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1364,
                          "end": 1375
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 1384,
                            "end": 1396
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1384,
                          "end": 1396
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1405,
                            "end": 1409
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1405,
                          "end": 1409
                        }
                      }
                    ],
                    "loc": {
                      "start": 1326,
                      "end": 1415
                    }
                  },
                  "loc": {
                    "start": 1313,
                    "end": 1415
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1420,
                      "end": 1422
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1420,
                    "end": 1422
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1427,
                      "end": 1437
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1427,
                    "end": 1437
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1442,
                      "end": 1452
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1442,
                    "end": 1452
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 1457,
                      "end": 1468
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1457,
                    "end": 1468
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 1473,
                      "end": 1486
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1473,
                    "end": 1486
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 1491,
                      "end": 1501
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1491,
                    "end": 1501
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 1506,
                      "end": 1515
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1506,
                    "end": 1515
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 1520,
                      "end": 1528
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1520,
                    "end": 1528
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1533,
                      "end": 1542
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1533,
                    "end": 1542
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 1547,
                      "end": 1557
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1547,
                    "end": 1557
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesStarted",
                    "loc": {
                      "start": 1562,
                      "end": 1574
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1562,
                    "end": 1574
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesCompleted",
                    "loc": {
                      "start": 1579,
                      "end": 1593
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1579,
                    "end": 1593
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "smartContractCallData",
                    "loc": {
                      "start": 1598,
                      "end": 1619
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1598,
                    "end": 1619
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apiCallData",
                    "loc": {
                      "start": 1624,
                      "end": 1635
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1624,
                    "end": 1635
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 1640,
                      "end": 1652
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1640,
                    "end": 1652
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 1657,
                      "end": 1669
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1657,
                    "end": 1669
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 1674,
                      "end": 1687
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1674,
                    "end": 1687
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 1692,
                      "end": 1714
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1692,
                    "end": 1714
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 1719,
                      "end": 1729
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1719,
                    "end": 1729
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 1734,
                      "end": 1745
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1734,
                    "end": 1745
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodesCount",
                    "loc": {
                      "start": 1750,
                      "end": 1760
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1750,
                    "end": 1760
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodeLinksCount",
                    "loc": {
                      "start": 1765,
                      "end": 1779
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1765,
                    "end": 1779
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "outputsCount",
                    "loc": {
                      "start": 1784,
                      "end": 1796
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1784,
                    "end": 1796
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 1801,
                      "end": 1813
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1801,
                    "end": 1813
                  }
                }
              ],
              "loc": {
                "start": 1307,
                "end": 1815
              }
            },
            "loc": {
              "start": 1298,
              "end": 1815
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1816,
                "end": 1818
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1816,
              "end": 1818
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1819,
                "end": 1829
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1819,
              "end": 1829
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1830,
                "end": 1840
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1830,
              "end": 1840
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 1841,
                "end": 1851
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1841,
              "end": 1851
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1852,
                "end": 1861
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1852,
              "end": 1861
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 1862,
                "end": 1873
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1862,
              "end": 1873
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 1874,
                "end": 1880
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
                      "start": 1890,
                      "end": 1900
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1887,
                    "end": 1900
                  }
                }
              ],
              "loc": {
                "start": 1881,
                "end": 1902
              }
            },
            "loc": {
              "start": 1874,
              "end": 1902
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1903,
                "end": 1908
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
                        "start": 1922,
                        "end": 1934
                      }
                    },
                    "loc": {
                      "start": 1922,
                      "end": 1934
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
                            "start": 1948,
                            "end": 1964
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1945,
                          "end": 1964
                        }
                      }
                    ],
                    "loc": {
                      "start": 1935,
                      "end": 1970
                    }
                  },
                  "loc": {
                    "start": 1915,
                    "end": 1970
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
                        "start": 1982,
                        "end": 1986
                      }
                    },
                    "loc": {
                      "start": 1982,
                      "end": 1986
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
                            "start": 2000,
                            "end": 2008
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1997,
                          "end": 2008
                        }
                      }
                    ],
                    "loc": {
                      "start": 1987,
                      "end": 2014
                    }
                  },
                  "loc": {
                    "start": 1975,
                    "end": 2014
                  }
                }
              ],
              "loc": {
                "start": 1909,
                "end": 2016
              }
            },
            "loc": {
              "start": 1903,
              "end": 2016
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 2017,
                "end": 2028
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2017,
              "end": 2028
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 2029,
                "end": 2043
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2029,
              "end": 2043
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 2044,
                "end": 2049
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2044,
              "end": 2049
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2050,
                "end": 2059
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2050,
              "end": 2059
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 2060,
                "end": 2064
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
                      "start": 2074,
                      "end": 2082
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2071,
                    "end": 2082
                  }
                }
              ],
              "loc": {
                "start": 2065,
                "end": 2084
              }
            },
            "loc": {
              "start": 2060,
              "end": 2084
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 2085,
                "end": 2099
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2085,
              "end": 2099
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 2100,
                "end": 2105
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2100,
              "end": 2105
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2106,
                "end": 2109
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
                      "start": 2116,
                      "end": 2126
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2116,
                    "end": 2126
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2131,
                      "end": 2140
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2131,
                    "end": 2140
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2145,
                      "end": 2156
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2145,
                    "end": 2156
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2161,
                      "end": 2170
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2161,
                    "end": 2170
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2175,
                      "end": 2182
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2175,
                    "end": 2182
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 2187,
                      "end": 2195
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2187,
                    "end": 2195
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2200,
                      "end": 2212
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2200,
                    "end": 2212
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2217,
                      "end": 2225
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2217,
                    "end": 2225
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 2230,
                      "end": 2238
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2230,
                    "end": 2238
                  }
                }
              ],
              "loc": {
                "start": 2110,
                "end": 2240
              }
            },
            "loc": {
              "start": 2106,
              "end": 2240
            }
          }
        ],
        "loc": {
          "start": 1296,
          "end": 2242
        }
      },
      "loc": {
        "start": 1263,
        "end": 2242
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 2252,
          "end": 2260
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 2264,
            "end": 2267
          }
        },
        "loc": {
          "start": 2264,
          "end": 2267
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
                "start": 2270,
                "end": 2272
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2270,
              "end": 2272
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2273,
                "end": 2283
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2273,
              "end": 2283
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 2284,
                "end": 2287
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2284,
              "end": 2287
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2288,
                "end": 2297
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2288,
              "end": 2297
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 2298,
                "end": 2310
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
                      "start": 2317,
                      "end": 2319
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2317,
                    "end": 2319
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2324,
                      "end": 2332
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2324,
                    "end": 2332
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2337,
                      "end": 2348
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2337,
                    "end": 2348
                  }
                }
              ],
              "loc": {
                "start": 2311,
                "end": 2350
              }
            },
            "loc": {
              "start": 2298,
              "end": 2350
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2351,
                "end": 2354
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
                      "start": 2361,
                      "end": 2366
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2361,
                    "end": 2366
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2371,
                      "end": 2383
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2371,
                    "end": 2383
                  }
                }
              ],
              "loc": {
                "start": 2355,
                "end": 2385
              }
            },
            "loc": {
              "start": 2351,
              "end": 2385
            }
          }
        ],
        "loc": {
          "start": 2268,
          "end": 2387
        }
      },
      "loc": {
        "start": 2243,
        "end": 2387
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 2397,
          "end": 2405
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 2409,
            "end": 2413
          }
        },
        "loc": {
          "start": 2409,
          "end": 2413
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
                "start": 2416,
                "end": 2418
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2416,
              "end": 2418
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 2419,
                "end": 2430
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2419,
              "end": 2430
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2431,
                "end": 2437
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2431,
              "end": 2437
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 2438,
                "end": 2443
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2438,
              "end": 2443
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 2444,
                "end": 2448
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2444,
              "end": 2448
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 2449,
                "end": 2461
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2449,
              "end": 2461
            }
          }
        ],
        "loc": {
          "start": 2414,
          "end": 2463
        }
      },
      "loc": {
        "start": 2388,
        "end": 2463
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
        "start": 2471,
        "end": 2488
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
              "start": 2490,
              "end": 2495
            }
          },
          "loc": {
            "start": 2489,
            "end": 2495
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
                "start": 2497,
                "end": 2524
              }
            },
            "loc": {
              "start": 2497,
              "end": 2524
            }
          },
          "loc": {
            "start": 2497,
            "end": 2525
          }
        },
        "directives": [],
        "loc": {
          "start": 2489,
          "end": 2525
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
              "start": 2531,
              "end": 2548
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 2549,
                  "end": 2554
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 2557,
                    "end": 2562
                  }
                },
                "loc": {
                  "start": 2556,
                  "end": 2562
                }
              },
              "loc": {
                "start": 2549,
                "end": 2562
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
                    "start": 2570,
                    "end": 2575
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
                          "start": 2586,
                          "end": 2592
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2586,
                        "end": 2592
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 2601,
                          "end": 2605
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
                                  "start": 2627,
                                  "end": 2634
                                }
                              },
                              "loc": {
                                "start": 2627,
                                "end": 2634
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
                                      "start": 2656,
                                      "end": 2668
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 2653,
                                    "end": 2668
                                  }
                                }
                              ],
                              "loc": {
                                "start": 2635,
                                "end": 2682
                              }
                            },
                            "loc": {
                              "start": 2620,
                              "end": 2682
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
                                  "start": 2702,
                                  "end": 2709
                                }
                              },
                              "loc": {
                                "start": 2702,
                                "end": 2709
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
                                      "start": 2731,
                                      "end": 2743
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 2728,
                                    "end": 2743
                                  }
                                }
                              ],
                              "loc": {
                                "start": 2710,
                                "end": 2757
                              }
                            },
                            "loc": {
                              "start": 2695,
                              "end": 2757
                            }
                          }
                        ],
                        "loc": {
                          "start": 2606,
                          "end": 2767
                        }
                      },
                      "loc": {
                        "start": 2601,
                        "end": 2767
                      }
                    }
                  ],
                  "loc": {
                    "start": 2576,
                    "end": 2773
                  }
                },
                "loc": {
                  "start": 2570,
                  "end": 2773
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 2778,
                    "end": 2786
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
                          "start": 2797,
                          "end": 2808
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2797,
                        "end": 2808
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorProject",
                        "loc": {
                          "start": 2817,
                          "end": 2833
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2817,
                        "end": 2833
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorRoutine",
                        "loc": {
                          "start": 2842,
                          "end": 2858
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2842,
                        "end": 2858
                      }
                    }
                  ],
                  "loc": {
                    "start": 2787,
                    "end": 2864
                  }
                },
                "loc": {
                  "start": 2778,
                  "end": 2864
                }
              }
            ],
            "loc": {
              "start": 2564,
              "end": 2868
            }
          },
          "loc": {
            "start": 2531,
            "end": 2868
          }
        }
      ],
      "loc": {
        "start": 2527,
        "end": 2870
      }
    },
    "loc": {
      "start": 2465,
      "end": 2870
    }
  },
  "variableValues": {},
  "path": {
    "key": "projectOrRoutine_findMany"
  }
} as const;
