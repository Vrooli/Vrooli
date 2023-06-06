export const runProjectOrRunRoutine_findMany = {
  "fieldName": "runProjectOrRunRoutines",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "runProjectOrRunRoutines",
        "loc": {
          "start": 2643,
          "end": 2666
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 2667,
              "end": 2672
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 2675,
                "end": 2680
              }
            },
            "loc": {
              "start": 2674,
              "end": 2680
            }
          },
          "loc": {
            "start": 2667,
            "end": 2680
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
                "start": 2688,
                "end": 2693
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
                      "start": 2704,
                      "end": 2710
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2704,
                    "end": 2710
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 2719,
                      "end": 2723
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
                            "value": "RunProject",
                            "loc": {
                              "start": 2745,
                              "end": 2755
                            }
                          },
                          "loc": {
                            "start": 2745,
                            "end": 2755
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
                                "value": "RunProject_list",
                                "loc": {
                                  "start": 2777,
                                  "end": 2792
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 2774,
                                "end": 2792
                              }
                            }
                          ],
                          "loc": {
                            "start": 2756,
                            "end": 2806
                          }
                        },
                        "loc": {
                          "start": 2738,
                          "end": 2806
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "RunRoutine",
                            "loc": {
                              "start": 2826,
                              "end": 2836
                            }
                          },
                          "loc": {
                            "start": 2826,
                            "end": 2836
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
                                "value": "RunRoutine_list",
                                "loc": {
                                  "start": 2858,
                                  "end": 2873
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 2855,
                                "end": 2873
                              }
                            }
                          ],
                          "loc": {
                            "start": 2837,
                            "end": 2887
                          }
                        },
                        "loc": {
                          "start": 2819,
                          "end": 2887
                        }
                      }
                    ],
                    "loc": {
                      "start": 2724,
                      "end": 2897
                    }
                  },
                  "loc": {
                    "start": 2719,
                    "end": 2897
                  }
                }
              ],
              "loc": {
                "start": 2694,
                "end": 2903
              }
            },
            "loc": {
              "start": 2688,
              "end": 2903
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 2908,
                "end": 2916
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
                      "start": 2927,
                      "end": 2938
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2927,
                    "end": 2938
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorRunProject",
                    "loc": {
                      "start": 2947,
                      "end": 2966
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2947,
                    "end": 2966
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorRunRoutine",
                    "loc": {
                      "start": 2975,
                      "end": 2994
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2975,
                    "end": 2994
                  }
                }
              ],
              "loc": {
                "start": 2917,
                "end": 3000
              }
            },
            "loc": {
              "start": 2908,
              "end": 3000
            }
          }
        ],
        "loc": {
          "start": 2682,
          "end": 3004
        }
      },
      "loc": {
        "start": 2643,
        "end": 3004
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "apisCount",
        "loc": {
          "start": 32,
          "end": 41
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 32,
        "end": 41
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "focusModesCount",
        "loc": {
          "start": 42,
          "end": 57
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 42,
        "end": 57
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 58,
          "end": 69
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 58,
        "end": 69
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "meetingsCount",
        "loc": {
          "start": 70,
          "end": 83
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 70,
        "end": 83
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "notesCount",
        "loc": {
          "start": 84,
          "end": 94
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 84,
        "end": 94
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "projectsCount",
        "loc": {
          "start": 95,
          "end": 108
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 95,
        "end": 108
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "routinesCount",
        "loc": {
          "start": 109,
          "end": 122
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 109,
        "end": 122
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "schedulesCount",
        "loc": {
          "start": 123,
          "end": 137
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 123,
        "end": 137
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "smartContractsCount",
        "loc": {
          "start": 138,
          "end": 157
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 138,
        "end": 157
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "standardsCount",
        "loc": {
          "start": 158,
          "end": 172
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 158,
        "end": 172
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 173,
          "end": 175
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 173,
        "end": 175
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 176,
          "end": 186
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 176,
        "end": 186
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 187,
          "end": 197
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 187,
        "end": 197
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 198,
          "end": 203
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 198,
        "end": 203
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 204,
          "end": 209
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 204,
        "end": 209
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 210,
          "end": 215
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
                  "start": 229,
                  "end": 241
                }
              },
              "loc": {
                "start": 229,
                "end": 241
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
                      "start": 255,
                      "end": 271
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 252,
                    "end": 271
                  }
                }
              ],
              "loc": {
                "start": 242,
                "end": 277
              }
            },
            "loc": {
              "start": 222,
              "end": 277
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
                  "start": 289,
                  "end": 293
                }
              },
              "loc": {
                "start": 289,
                "end": 293
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
                      "start": 307,
                      "end": 315
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 304,
                    "end": 315
                  }
                }
              ],
              "loc": {
                "start": 294,
                "end": 321
              }
            },
            "loc": {
              "start": 282,
              "end": 321
            }
          }
        ],
        "loc": {
          "start": 216,
          "end": 323
        }
      },
      "loc": {
        "start": 210,
        "end": 323
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 324,
          "end": 327
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
                "start": 334,
                "end": 343
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 334,
              "end": 343
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 348,
                "end": 357
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 348,
              "end": 357
            }
          }
        ],
        "loc": {
          "start": 328,
          "end": 359
        }
      },
      "loc": {
        "start": 324,
        "end": 359
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 406,
          "end": 408
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 406,
        "end": 408
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 409,
          "end": 415
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 409,
        "end": 415
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 416,
          "end": 419
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
                "start": 426,
                "end": 439
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 426,
              "end": 439
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 444,
                "end": 453
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 444,
              "end": 453
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 458,
                "end": 469
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 458,
              "end": 469
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 474,
                "end": 483
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 474,
              "end": 483
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 488,
                "end": 497
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 488,
              "end": 497
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
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
              "value": "isBookmarked",
              "loc": {
                "start": 514,
                "end": 526
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 514,
              "end": 526
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 531,
                "end": 539
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 531,
              "end": 539
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 544,
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
                    "value": "id",
                    "loc": {
                      "start": 569,
                      "end": 571
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 569,
                    "end": 571
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 580,
                      "end": 590
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 580,
                    "end": 590
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 599,
                      "end": 609
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 599,
                    "end": 609
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 618,
                      "end": 625
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 618,
                    "end": 625
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 634,
                      "end": 645
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 634,
                    "end": 645
                  }
                }
              ],
              "loc": {
                "start": 559,
                "end": 651
              }
            },
            "loc": {
              "start": 544,
              "end": 651
            }
          }
        ],
        "loc": {
          "start": 420,
          "end": 653
        }
      },
      "loc": {
        "start": 416,
        "end": 653
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "projectVersion",
        "loc": {
          "start": 697,
          "end": 711
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
                "start": 718,
                "end": 720
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 718,
              "end": 720
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "complexity",
              "loc": {
                "start": 725,
                "end": 735
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 725,
              "end": 735
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 740,
                "end": 748
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 740,
              "end": 748
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 753,
                "end": 762
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 753,
              "end": 762
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 767,
                "end": 779
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 767,
              "end": 779
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 784,
                "end": 796
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 784,
              "end": 796
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "root",
              "loc": {
                "start": 801,
                "end": 805
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
                      "start": 816,
                      "end": 818
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 816,
                    "end": 818
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 827,
                      "end": 836
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 827,
                    "end": 836
                  }
                }
              ],
              "loc": {
                "start": 806,
                "end": 842
              }
            },
            "loc": {
              "start": 801,
              "end": 842
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 847,
                "end": 859
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
                      "start": 870,
                      "end": 872
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 870,
                    "end": 872
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 881,
                      "end": 889
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 881,
                    "end": 889
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 898,
                      "end": 909
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 898,
                    "end": 909
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 918,
                      "end": 922
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 918,
                    "end": 922
                  }
                }
              ],
              "loc": {
                "start": 860,
                "end": 928
              }
            },
            "loc": {
              "start": 847,
              "end": 928
            }
          }
        ],
        "loc": {
          "start": 712,
          "end": 930
        }
      },
      "loc": {
        "start": 697,
        "end": 930
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 931,
          "end": 933
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 931,
        "end": 933
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 934,
          "end": 943
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 934,
        "end": 943
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "completedComplexity",
        "loc": {
          "start": 944,
          "end": 963
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 944,
        "end": 963
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "contextSwitches",
        "loc": {
          "start": 964,
          "end": 979
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 964,
        "end": 979
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startedAt",
        "loc": {
          "start": 980,
          "end": 989
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 980,
        "end": 989
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timeElapsed",
        "loc": {
          "start": 990,
          "end": 1001
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 990,
        "end": 1001
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "completedAt",
        "loc": {
          "start": 1002,
          "end": 1013
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1002,
        "end": 1013
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 1014,
          "end": 1018
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1014,
        "end": 1018
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "status",
        "loc": {
          "start": 1019,
          "end": 1025
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1019,
        "end": 1025
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "stepsCount",
        "loc": {
          "start": 1026,
          "end": 1036
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1026,
        "end": 1036
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "organization",
        "loc": {
          "start": 1037,
          "end": 1049
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
              "value": "Organization_nav",
              "loc": {
                "start": 1059,
                "end": 1075
              }
            },
            "directives": [],
            "loc": {
              "start": 1056,
              "end": 1075
            }
          }
        ],
        "loc": {
          "start": 1050,
          "end": 1077
        }
      },
      "loc": {
        "start": 1037,
        "end": 1077
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "schedule",
        "loc": {
          "start": 1078,
          "end": 1086
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
              "value": "labels",
              "loc": {
                "start": 1093,
                "end": 1099
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
                    "value": "Label_full",
                    "loc": {
                      "start": 1113,
                      "end": 1123
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1110,
                    "end": 1123
                  }
                }
              ],
              "loc": {
                "start": 1100,
                "end": 1129
              }
            },
            "loc": {
              "start": 1093,
              "end": 1129
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1134,
                "end": 1136
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1134,
              "end": 1136
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1141,
                "end": 1151
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1141,
              "end": 1151
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1156,
                "end": 1166
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1156,
              "end": 1166
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 1171,
                "end": 1180
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1171,
              "end": 1180
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 1185,
                "end": 1192
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1185,
              "end": 1192
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 1197,
                "end": 1205
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1197,
              "end": 1205
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 1210,
                "end": 1220
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
                      "start": 1231,
                      "end": 1233
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1231,
                    "end": 1233
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 1242,
                      "end": 1259
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1242,
                    "end": 1259
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 1268,
                      "end": 1280
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1268,
                    "end": 1280
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 1289,
                      "end": 1299
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1289,
                    "end": 1299
                  }
                }
              ],
              "loc": {
                "start": 1221,
                "end": 1305
              }
            },
            "loc": {
              "start": 1210,
              "end": 1305
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 1310,
                "end": 1321
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
                      "start": 1332,
                      "end": 1334
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1332,
                    "end": 1334
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 1343,
                      "end": 1357
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1343,
                    "end": 1357
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 1366,
                      "end": 1374
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1366,
                    "end": 1374
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 1383,
                      "end": 1392
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1383,
                    "end": 1392
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 1401,
                      "end": 1411
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1401,
                    "end": 1411
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 1420,
                      "end": 1425
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1420,
                    "end": 1425
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 1434,
                      "end": 1441
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1434,
                    "end": 1441
                  }
                }
              ],
              "loc": {
                "start": 1322,
                "end": 1447
              }
            },
            "loc": {
              "start": 1310,
              "end": 1447
            }
          }
        ],
        "loc": {
          "start": 1087,
          "end": 1449
        }
      },
      "loc": {
        "start": 1078,
        "end": 1449
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "user",
        "loc": {
          "start": 1450,
          "end": 1454
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
              "value": "User_nav",
              "loc": {
                "start": 1464,
                "end": 1472
              }
            },
            "directives": [],
            "loc": {
              "start": 1461,
              "end": 1472
            }
          }
        ],
        "loc": {
          "start": 1455,
          "end": 1474
        }
      },
      "loc": {
        "start": 1450,
        "end": 1474
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1475,
          "end": 1478
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
                "start": 1485,
                "end": 1494
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1485,
              "end": 1494
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1499,
                "end": 1508
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1499,
              "end": 1508
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1513,
                "end": 1520
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1513,
              "end": 1520
            }
          }
        ],
        "loc": {
          "start": 1479,
          "end": 1522
        }
      },
      "loc": {
        "start": 1475,
        "end": 1522
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "routineVersion",
        "loc": {
          "start": 1566,
          "end": 1580
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
                "start": 1587,
                "end": 1589
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1587,
              "end": 1589
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "complexity",
              "loc": {
                "start": 1594,
                "end": 1604
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1594,
              "end": 1604
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isAutomatable",
              "loc": {
                "start": 1609,
                "end": 1622
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1609,
              "end": 1622
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 1627,
                "end": 1637
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1627,
              "end": 1637
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 1642,
                "end": 1651
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1642,
              "end": 1651
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 1656,
                "end": 1664
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1656,
              "end": 1664
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1669,
                "end": 1678
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1669,
              "end": 1678
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "root",
              "loc": {
                "start": 1683,
                "end": 1687
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
                      "start": 1698,
                      "end": 1700
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1698,
                    "end": 1700
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isInternal",
                    "loc": {
                      "start": 1709,
                      "end": 1719
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1709,
                    "end": 1719
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1728,
                      "end": 1737
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1728,
                    "end": 1737
                  }
                }
              ],
              "loc": {
                "start": 1688,
                "end": 1743
              }
            },
            "loc": {
              "start": 1683,
              "end": 1743
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1748,
                "end": 1760
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
                      "start": 1771,
                      "end": 1773
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1771,
                    "end": 1773
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1782,
                      "end": 1790
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1782,
                    "end": 1790
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1799,
                      "end": 1810
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1799,
                    "end": 1810
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "instructions",
                    "loc": {
                      "start": 1819,
                      "end": 1831
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1819,
                    "end": 1831
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1840,
                      "end": 1844
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1840,
                    "end": 1844
                  }
                }
              ],
              "loc": {
                "start": 1761,
                "end": 1850
              }
            },
            "loc": {
              "start": 1748,
              "end": 1850
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 1855,
                "end": 1867
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1855,
              "end": 1867
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 1872,
                "end": 1884
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1872,
              "end": 1884
            }
          }
        ],
        "loc": {
          "start": 1581,
          "end": 1886
        }
      },
      "loc": {
        "start": 1566,
        "end": 1886
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1887,
          "end": 1889
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1887,
        "end": 1889
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 1890,
          "end": 1899
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1890,
        "end": 1899
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "completedComplexity",
        "loc": {
          "start": 1900,
          "end": 1919
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1900,
        "end": 1919
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "contextSwitches",
        "loc": {
          "start": 1920,
          "end": 1935
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1920,
        "end": 1935
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startedAt",
        "loc": {
          "start": 1936,
          "end": 1945
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1936,
        "end": 1945
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timeElapsed",
        "loc": {
          "start": 1946,
          "end": 1957
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1946,
        "end": 1957
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "completedAt",
        "loc": {
          "start": 1958,
          "end": 1969
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1958,
        "end": 1969
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 1970,
          "end": 1974
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1970,
        "end": 1974
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "status",
        "loc": {
          "start": 1975,
          "end": 1981
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1975,
        "end": 1981
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "stepsCount",
        "loc": {
          "start": 1982,
          "end": 1992
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1982,
        "end": 1992
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "inputsCount",
        "loc": {
          "start": 1993,
          "end": 2004
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1993,
        "end": 2004
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "wasRunAutomatically",
        "loc": {
          "start": 2005,
          "end": 2024
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2005,
        "end": 2024
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "organization",
        "loc": {
          "start": 2025,
          "end": 2037
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
              "value": "Organization_nav",
              "loc": {
                "start": 2047,
                "end": 2063
              }
            },
            "directives": [],
            "loc": {
              "start": 2044,
              "end": 2063
            }
          }
        ],
        "loc": {
          "start": 2038,
          "end": 2065
        }
      },
      "loc": {
        "start": 2025,
        "end": 2065
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "schedule",
        "loc": {
          "start": 2066,
          "end": 2074
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
              "value": "labels",
              "loc": {
                "start": 2081,
                "end": 2087
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
                    "value": "Label_full",
                    "loc": {
                      "start": 2101,
                      "end": 2111
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2098,
                    "end": 2111
                  }
                }
              ],
              "loc": {
                "start": 2088,
                "end": 2117
              }
            },
            "loc": {
              "start": 2081,
              "end": 2117
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2122,
                "end": 2124
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2122,
              "end": 2124
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2129,
                "end": 2139
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2129,
              "end": 2139
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2144,
                "end": 2154
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2144,
              "end": 2154
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 2159,
                "end": 2168
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2159,
              "end": 2168
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 2173,
                "end": 2180
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2173,
              "end": 2180
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 2185,
                "end": 2193
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2185,
              "end": 2193
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 2198,
                "end": 2208
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
                      "start": 2219,
                      "end": 2221
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2219,
                    "end": 2221
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 2230,
                      "end": 2247
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2230,
                    "end": 2247
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 2256,
                      "end": 2268
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2256,
                    "end": 2268
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 2277,
                      "end": 2287
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2277,
                    "end": 2287
                  }
                }
              ],
              "loc": {
                "start": 2209,
                "end": 2293
              }
            },
            "loc": {
              "start": 2198,
              "end": 2293
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 2298,
                "end": 2309
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
                      "start": 2320,
                      "end": 2322
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2320,
                    "end": 2322
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 2331,
                      "end": 2345
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2331,
                    "end": 2345
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 2354,
                      "end": 2362
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2354,
                    "end": 2362
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 2371,
                      "end": 2380
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2371,
                    "end": 2380
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 2389,
                      "end": 2399
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2389,
                    "end": 2399
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 2408,
                      "end": 2413
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2408,
                    "end": 2413
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 2422,
                      "end": 2429
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2422,
                    "end": 2429
                  }
                }
              ],
              "loc": {
                "start": 2310,
                "end": 2435
              }
            },
            "loc": {
              "start": 2298,
              "end": 2435
            }
          }
        ],
        "loc": {
          "start": 2075,
          "end": 2437
        }
      },
      "loc": {
        "start": 2066,
        "end": 2437
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "user",
        "loc": {
          "start": 2438,
          "end": 2442
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
              "value": "User_nav",
              "loc": {
                "start": 2452,
                "end": 2460
              }
            },
            "directives": [],
            "loc": {
              "start": 2449,
              "end": 2460
            }
          }
        ],
        "loc": {
          "start": 2443,
          "end": 2462
        }
      },
      "loc": {
        "start": 2438,
        "end": 2462
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2463,
          "end": 2466
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
                "start": 2473,
                "end": 2482
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2473,
              "end": 2482
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2487,
                "end": 2496
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2487,
              "end": 2496
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 2501,
                "end": 2508
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2501,
              "end": 2508
            }
          }
        ],
        "loc": {
          "start": 2467,
          "end": 2510
        }
      },
      "loc": {
        "start": 2463,
        "end": 2510
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2541,
          "end": 2543
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2541,
        "end": 2543
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 2544,
          "end": 2549
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2544,
        "end": 2549
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 2550,
          "end": 2554
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2550,
        "end": 2554
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 2555,
          "end": 2561
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2555,
        "end": 2561
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Label_full": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_full",
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
              "value": "apisCount",
              "loc": {
                "start": 32,
                "end": 41
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 32,
              "end": 41
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusModesCount",
              "loc": {
                "start": 42,
                "end": 57
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 42,
              "end": 57
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 58,
                "end": 69
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 58,
              "end": 69
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "meetingsCount",
              "loc": {
                "start": 70,
                "end": 83
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 70,
              "end": 83
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "notesCount",
              "loc": {
                "start": 84,
                "end": 94
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 84,
              "end": 94
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "projectsCount",
              "loc": {
                "start": 95,
                "end": 108
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 95,
              "end": 108
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "routinesCount",
              "loc": {
                "start": 109,
                "end": 122
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 109,
              "end": 122
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "schedulesCount",
              "loc": {
                "start": 123,
                "end": 137
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 123,
              "end": 137
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "smartContractsCount",
              "loc": {
                "start": 138,
                "end": 157
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 138,
              "end": 157
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "standardsCount",
              "loc": {
                "start": 158,
                "end": 172
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 158,
              "end": 172
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 173,
                "end": 175
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 173,
              "end": 175
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 176,
                "end": 186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 176,
              "end": 186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 187,
                "end": 197
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 187,
              "end": 197
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 198,
                "end": 203
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 198,
              "end": 203
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 204,
                "end": 209
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 204,
              "end": 209
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 210,
                "end": 215
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
                        "start": 229,
                        "end": 241
                      }
                    },
                    "loc": {
                      "start": 229,
                      "end": 241
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
                            "start": 255,
                            "end": 271
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 252,
                          "end": 271
                        }
                      }
                    ],
                    "loc": {
                      "start": 242,
                      "end": 277
                    }
                  },
                  "loc": {
                    "start": 222,
                    "end": 277
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
                        "start": 289,
                        "end": 293
                      }
                    },
                    "loc": {
                      "start": 289,
                      "end": 293
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
                            "start": 307,
                            "end": 315
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 304,
                          "end": 315
                        }
                      }
                    ],
                    "loc": {
                      "start": 294,
                      "end": 321
                    }
                  },
                  "loc": {
                    "start": 282,
                    "end": 321
                  }
                }
              ],
              "loc": {
                "start": 216,
                "end": 323
              }
            },
            "loc": {
              "start": 210,
              "end": 323
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 324,
                "end": 327
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
                      "start": 334,
                      "end": 343
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 334,
                    "end": 343
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 348,
                      "end": 357
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 348,
                    "end": 357
                  }
                }
              ],
              "loc": {
                "start": 328,
                "end": 359
              }
            },
            "loc": {
              "start": 324,
              "end": 359
            }
          }
        ],
        "loc": {
          "start": 30,
          "end": 361
        }
      },
      "loc": {
        "start": 1,
        "end": 361
      }
    },
    "Organization_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_nav",
        "loc": {
          "start": 371,
          "end": 387
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 391,
            "end": 403
          }
        },
        "loc": {
          "start": 391,
          "end": 403
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
                "start": 406,
                "end": 408
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 406,
              "end": 408
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 409,
                "end": 415
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 409,
              "end": 415
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 416,
                "end": 419
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
                      "start": 426,
                      "end": 439
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 426,
                    "end": 439
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 444,
                      "end": 453
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 444,
                    "end": 453
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 458,
                      "end": 469
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 458,
                    "end": 469
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 474,
                      "end": 483
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 474,
                    "end": 483
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 488,
                      "end": 497
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 488,
                    "end": 497
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
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
                    "value": "isBookmarked",
                    "loc": {
                      "start": 514,
                      "end": 526
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 514,
                    "end": 526
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 531,
                      "end": 539
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 531,
                    "end": 539
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 544,
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
                          "value": "id",
                          "loc": {
                            "start": 569,
                            "end": 571
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 569,
                          "end": 571
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 580,
                            "end": 590
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 580,
                          "end": 590
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 599,
                            "end": 609
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 599,
                          "end": 609
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 618,
                            "end": 625
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 618,
                          "end": 625
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 634,
                            "end": 645
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 634,
                          "end": 645
                        }
                      }
                    ],
                    "loc": {
                      "start": 559,
                      "end": 651
                    }
                  },
                  "loc": {
                    "start": 544,
                    "end": 651
                  }
                }
              ],
              "loc": {
                "start": 420,
                "end": 653
              }
            },
            "loc": {
              "start": 416,
              "end": 653
            }
          }
        ],
        "loc": {
          "start": 404,
          "end": 655
        }
      },
      "loc": {
        "start": 362,
        "end": 655
      }
    },
    "RunProject_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "RunProject_list",
        "loc": {
          "start": 665,
          "end": 680
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "RunProject",
          "loc": {
            "start": 684,
            "end": 694
          }
        },
        "loc": {
          "start": 684,
          "end": 694
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
              "value": "projectVersion",
              "loc": {
                "start": 697,
                "end": 711
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
                      "start": 718,
                      "end": 720
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 718,
                    "end": 720
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "complexity",
                    "loc": {
                      "start": 725,
                      "end": 735
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 725,
                    "end": 735
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 740,
                      "end": 748
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 740,
                    "end": 748
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 753,
                      "end": 762
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 753,
                    "end": 762
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 767,
                      "end": 779
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 767,
                    "end": 779
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 784,
                      "end": 796
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 784,
                    "end": 796
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "root",
                    "loc": {
                      "start": 801,
                      "end": 805
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
                            "start": 816,
                            "end": 818
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 816,
                          "end": 818
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 827,
                            "end": 836
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 827,
                          "end": 836
                        }
                      }
                    ],
                    "loc": {
                      "start": 806,
                      "end": 842
                    }
                  },
                  "loc": {
                    "start": 801,
                    "end": 842
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 847,
                      "end": 859
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
                            "start": 870,
                            "end": 872
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 870,
                          "end": 872
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 881,
                            "end": 889
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 881,
                          "end": 889
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 898,
                            "end": 909
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 898,
                          "end": 909
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 918,
                            "end": 922
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 918,
                          "end": 922
                        }
                      }
                    ],
                    "loc": {
                      "start": 860,
                      "end": 928
                    }
                  },
                  "loc": {
                    "start": 847,
                    "end": 928
                  }
                }
              ],
              "loc": {
                "start": 712,
                "end": 930
              }
            },
            "loc": {
              "start": 697,
              "end": 930
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 931,
                "end": 933
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 931,
              "end": 933
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 934,
                "end": 943
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 934,
              "end": 943
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedComplexity",
              "loc": {
                "start": 944,
                "end": 963
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 944,
              "end": 963
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contextSwitches",
              "loc": {
                "start": 964,
                "end": 979
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 964,
              "end": 979
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startedAt",
              "loc": {
                "start": 980,
                "end": 989
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 980,
              "end": 989
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeElapsed",
              "loc": {
                "start": 990,
                "end": 1001
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 990,
              "end": 1001
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 1002,
                "end": 1013
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1002,
              "end": 1013
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1014,
                "end": 1018
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1014,
              "end": 1018
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "status",
              "loc": {
                "start": 1019,
                "end": 1025
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1019,
              "end": 1025
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stepsCount",
              "loc": {
                "start": 1026,
                "end": 1036
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1026,
              "end": 1036
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 1037,
                "end": 1049
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
                    "value": "Organization_nav",
                    "loc": {
                      "start": 1059,
                      "end": 1075
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1056,
                    "end": 1075
                  }
                }
              ],
              "loc": {
                "start": 1050,
                "end": 1077
              }
            },
            "loc": {
              "start": 1037,
              "end": 1077
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "schedule",
              "loc": {
                "start": 1078,
                "end": 1086
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
                    "value": "labels",
                    "loc": {
                      "start": 1093,
                      "end": 1099
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
                          "value": "Label_full",
                          "loc": {
                            "start": 1113,
                            "end": 1123
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1110,
                          "end": 1123
                        }
                      }
                    ],
                    "loc": {
                      "start": 1100,
                      "end": 1129
                    }
                  },
                  "loc": {
                    "start": 1093,
                    "end": 1129
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1134,
                      "end": 1136
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1134,
                    "end": 1136
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1141,
                      "end": 1151
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1141,
                    "end": 1151
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1156,
                      "end": 1166
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1156,
                    "end": 1166
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "startTime",
                    "loc": {
                      "start": 1171,
                      "end": 1180
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1171,
                    "end": 1180
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endTime",
                    "loc": {
                      "start": 1185,
                      "end": 1192
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1185,
                    "end": 1192
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timezone",
                    "loc": {
                      "start": 1197,
                      "end": 1205
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1197,
                    "end": 1205
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "exceptions",
                    "loc": {
                      "start": 1210,
                      "end": 1220
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
                            "start": 1231,
                            "end": 1233
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1231,
                          "end": 1233
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "originalStartTime",
                          "loc": {
                            "start": 1242,
                            "end": 1259
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1242,
                          "end": 1259
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "newStartTime",
                          "loc": {
                            "start": 1268,
                            "end": 1280
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1268,
                          "end": 1280
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "newEndTime",
                          "loc": {
                            "start": 1289,
                            "end": 1299
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1289,
                          "end": 1299
                        }
                      }
                    ],
                    "loc": {
                      "start": 1221,
                      "end": 1305
                    }
                  },
                  "loc": {
                    "start": 1210,
                    "end": 1305
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrences",
                    "loc": {
                      "start": 1310,
                      "end": 1321
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
                            "start": 1332,
                            "end": 1334
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1332,
                          "end": 1334
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "recurrenceType",
                          "loc": {
                            "start": 1343,
                            "end": 1357
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1343,
                          "end": 1357
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "interval",
                          "loc": {
                            "start": 1366,
                            "end": 1374
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1366,
                          "end": 1374
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "dayOfWeek",
                          "loc": {
                            "start": 1383,
                            "end": 1392
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1383,
                          "end": 1392
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "dayOfMonth",
                          "loc": {
                            "start": 1401,
                            "end": 1411
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1401,
                          "end": 1411
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "month",
                          "loc": {
                            "start": 1420,
                            "end": 1425
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1420,
                          "end": 1425
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "endDate",
                          "loc": {
                            "start": 1434,
                            "end": 1441
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1434,
                          "end": 1441
                        }
                      }
                    ],
                    "loc": {
                      "start": 1322,
                      "end": 1447
                    }
                  },
                  "loc": {
                    "start": 1310,
                    "end": 1447
                  }
                }
              ],
              "loc": {
                "start": 1087,
                "end": 1449
              }
            },
            "loc": {
              "start": 1078,
              "end": 1449
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "user",
              "loc": {
                "start": 1450,
                "end": 1454
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
                    "value": "User_nav",
                    "loc": {
                      "start": 1464,
                      "end": 1472
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1461,
                    "end": 1472
                  }
                }
              ],
              "loc": {
                "start": 1455,
                "end": 1474
              }
            },
            "loc": {
              "start": 1450,
              "end": 1474
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1475,
                "end": 1478
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
                      "start": 1485,
                      "end": 1494
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1485,
                    "end": 1494
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1499,
                      "end": 1508
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1499,
                    "end": 1508
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1513,
                      "end": 1520
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1513,
                    "end": 1520
                  }
                }
              ],
              "loc": {
                "start": 1479,
                "end": 1522
              }
            },
            "loc": {
              "start": 1475,
              "end": 1522
            }
          }
        ],
        "loc": {
          "start": 695,
          "end": 1524
        }
      },
      "loc": {
        "start": 656,
        "end": 1524
      }
    },
    "RunRoutine_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "RunRoutine_list",
        "loc": {
          "start": 1534,
          "end": 1549
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "RunRoutine",
          "loc": {
            "start": 1553,
            "end": 1563
          }
        },
        "loc": {
          "start": 1553,
          "end": 1563
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
              "value": "routineVersion",
              "loc": {
                "start": 1566,
                "end": 1580
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
                      "start": 1587,
                      "end": 1589
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1587,
                    "end": 1589
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "complexity",
                    "loc": {
                      "start": 1594,
                      "end": 1604
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1594,
                    "end": 1604
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 1609,
                      "end": 1622
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1609,
                    "end": 1622
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 1627,
                      "end": 1637
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1627,
                    "end": 1637
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 1642,
                      "end": 1651
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1642,
                    "end": 1651
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 1656,
                      "end": 1664
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1656,
                    "end": 1664
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1669,
                      "end": 1678
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1669,
                    "end": 1678
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "root",
                    "loc": {
                      "start": 1683,
                      "end": 1687
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
                            "start": 1698,
                            "end": 1700
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1698,
                          "end": 1700
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isInternal",
                          "loc": {
                            "start": 1709,
                            "end": 1719
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1709,
                          "end": 1719
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 1728,
                            "end": 1737
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1728,
                          "end": 1737
                        }
                      }
                    ],
                    "loc": {
                      "start": 1688,
                      "end": 1743
                    }
                  },
                  "loc": {
                    "start": 1683,
                    "end": 1743
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 1748,
                      "end": 1760
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
                            "start": 1771,
                            "end": 1773
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1771,
                          "end": 1773
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 1782,
                            "end": 1790
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1782,
                          "end": 1790
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1799,
                            "end": 1810
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1799,
                          "end": 1810
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 1819,
                            "end": 1831
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1819,
                          "end": 1831
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1840,
                            "end": 1844
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1840,
                          "end": 1844
                        }
                      }
                    ],
                    "loc": {
                      "start": 1761,
                      "end": 1850
                    }
                  },
                  "loc": {
                    "start": 1748,
                    "end": 1850
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 1855,
                      "end": 1867
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1855,
                    "end": 1867
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 1872,
                      "end": 1884
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1872,
                    "end": 1884
                  }
                }
              ],
              "loc": {
                "start": 1581,
                "end": 1886
              }
            },
            "loc": {
              "start": 1566,
              "end": 1886
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1887,
                "end": 1889
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1887,
              "end": 1889
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1890,
                "end": 1899
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1890,
              "end": 1899
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedComplexity",
              "loc": {
                "start": 1900,
                "end": 1919
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1900,
              "end": 1919
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contextSwitches",
              "loc": {
                "start": 1920,
                "end": 1935
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1920,
              "end": 1935
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startedAt",
              "loc": {
                "start": 1936,
                "end": 1945
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1936,
              "end": 1945
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeElapsed",
              "loc": {
                "start": 1946,
                "end": 1957
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1946,
              "end": 1957
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 1958,
                "end": 1969
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1958,
              "end": 1969
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1970,
                "end": 1974
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1970,
              "end": 1974
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "status",
              "loc": {
                "start": 1975,
                "end": 1981
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1975,
              "end": 1981
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stepsCount",
              "loc": {
                "start": 1982,
                "end": 1992
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1982,
              "end": 1992
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 1993,
                "end": 2004
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1993,
              "end": 2004
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "wasRunAutomatically",
              "loc": {
                "start": 2005,
                "end": 2024
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2005,
              "end": 2024
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 2025,
                "end": 2037
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
                    "value": "Organization_nav",
                    "loc": {
                      "start": 2047,
                      "end": 2063
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2044,
                    "end": 2063
                  }
                }
              ],
              "loc": {
                "start": 2038,
                "end": 2065
              }
            },
            "loc": {
              "start": 2025,
              "end": 2065
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "schedule",
              "loc": {
                "start": 2066,
                "end": 2074
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
                    "value": "labels",
                    "loc": {
                      "start": 2081,
                      "end": 2087
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
                          "value": "Label_full",
                          "loc": {
                            "start": 2101,
                            "end": 2111
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2098,
                          "end": 2111
                        }
                      }
                    ],
                    "loc": {
                      "start": 2088,
                      "end": 2117
                    }
                  },
                  "loc": {
                    "start": 2081,
                    "end": 2117
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2122,
                      "end": 2124
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2122,
                    "end": 2124
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2129,
                      "end": 2139
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2129,
                    "end": 2139
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2144,
                      "end": 2154
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2144,
                    "end": 2154
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "startTime",
                    "loc": {
                      "start": 2159,
                      "end": 2168
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2159,
                    "end": 2168
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endTime",
                    "loc": {
                      "start": 2173,
                      "end": 2180
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2173,
                    "end": 2180
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timezone",
                    "loc": {
                      "start": 2185,
                      "end": 2193
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2185,
                    "end": 2193
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "exceptions",
                    "loc": {
                      "start": 2198,
                      "end": 2208
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
                            "start": 2219,
                            "end": 2221
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2219,
                          "end": 2221
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "originalStartTime",
                          "loc": {
                            "start": 2230,
                            "end": 2247
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2230,
                          "end": 2247
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "newStartTime",
                          "loc": {
                            "start": 2256,
                            "end": 2268
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2256,
                          "end": 2268
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "newEndTime",
                          "loc": {
                            "start": 2277,
                            "end": 2287
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2277,
                          "end": 2287
                        }
                      }
                    ],
                    "loc": {
                      "start": 2209,
                      "end": 2293
                    }
                  },
                  "loc": {
                    "start": 2198,
                    "end": 2293
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrences",
                    "loc": {
                      "start": 2298,
                      "end": 2309
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
                            "start": 2320,
                            "end": 2322
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2320,
                          "end": 2322
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "recurrenceType",
                          "loc": {
                            "start": 2331,
                            "end": 2345
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2331,
                          "end": 2345
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "interval",
                          "loc": {
                            "start": 2354,
                            "end": 2362
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2354,
                          "end": 2362
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "dayOfWeek",
                          "loc": {
                            "start": 2371,
                            "end": 2380
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2371,
                          "end": 2380
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "dayOfMonth",
                          "loc": {
                            "start": 2389,
                            "end": 2399
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2389,
                          "end": 2399
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "month",
                          "loc": {
                            "start": 2408,
                            "end": 2413
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2408,
                          "end": 2413
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "endDate",
                          "loc": {
                            "start": 2422,
                            "end": 2429
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2422,
                          "end": 2429
                        }
                      }
                    ],
                    "loc": {
                      "start": 2310,
                      "end": 2435
                    }
                  },
                  "loc": {
                    "start": 2298,
                    "end": 2435
                  }
                }
              ],
              "loc": {
                "start": 2075,
                "end": 2437
              }
            },
            "loc": {
              "start": 2066,
              "end": 2437
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "user",
              "loc": {
                "start": 2438,
                "end": 2442
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
                    "value": "User_nav",
                    "loc": {
                      "start": 2452,
                      "end": 2460
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2449,
                    "end": 2460
                  }
                }
              ],
              "loc": {
                "start": 2443,
                "end": 2462
              }
            },
            "loc": {
              "start": 2438,
              "end": 2462
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2463,
                "end": 2466
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
                      "start": 2473,
                      "end": 2482
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2473,
                    "end": 2482
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2487,
                      "end": 2496
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2487,
                    "end": 2496
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2501,
                      "end": 2508
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2501,
                    "end": 2508
                  }
                }
              ],
              "loc": {
                "start": 2467,
                "end": 2510
              }
            },
            "loc": {
              "start": 2463,
              "end": 2510
            }
          }
        ],
        "loc": {
          "start": 1564,
          "end": 2512
        }
      },
      "loc": {
        "start": 1525,
        "end": 2512
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 2522,
          "end": 2530
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 2534,
            "end": 2538
          }
        },
        "loc": {
          "start": 2534,
          "end": 2538
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
                "start": 2541,
                "end": 2543
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2541,
              "end": 2543
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 2544,
                "end": 2549
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2544,
              "end": 2549
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 2550,
                "end": 2554
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2550,
              "end": 2554
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2555,
                "end": 2561
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2555,
              "end": 2561
            }
          }
        ],
        "loc": {
          "start": 2539,
          "end": 2563
        }
      },
      "loc": {
        "start": 2513,
        "end": 2563
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "runProjectOrRunRoutines",
      "loc": {
        "start": 2571,
        "end": 2594
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
              "start": 2596,
              "end": 2601
            }
          },
          "loc": {
            "start": 2595,
            "end": 2601
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "RunProjectOrRunRoutineSearchInput",
              "loc": {
                "start": 2603,
                "end": 2636
              }
            },
            "loc": {
              "start": 2603,
              "end": 2636
            }
          },
          "loc": {
            "start": 2603,
            "end": 2637
          }
        },
        "directives": [],
        "loc": {
          "start": 2595,
          "end": 2637
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
            "value": "runProjectOrRunRoutines",
            "loc": {
              "start": 2643,
              "end": 2666
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 2667,
                  "end": 2672
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 2675,
                    "end": 2680
                  }
                },
                "loc": {
                  "start": 2674,
                  "end": 2680
                }
              },
              "loc": {
                "start": 2667,
                "end": 2680
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
                    "start": 2688,
                    "end": 2693
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
                          "start": 2704,
                          "end": 2710
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2704,
                        "end": 2710
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 2719,
                          "end": 2723
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
                                "value": "RunProject",
                                "loc": {
                                  "start": 2745,
                                  "end": 2755
                                }
                              },
                              "loc": {
                                "start": 2745,
                                "end": 2755
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
                                    "value": "RunProject_list",
                                    "loc": {
                                      "start": 2777,
                                      "end": 2792
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 2774,
                                    "end": 2792
                                  }
                                }
                              ],
                              "loc": {
                                "start": 2756,
                                "end": 2806
                              }
                            },
                            "loc": {
                              "start": 2738,
                              "end": 2806
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "RunRoutine",
                                "loc": {
                                  "start": 2826,
                                  "end": 2836
                                }
                              },
                              "loc": {
                                "start": 2826,
                                "end": 2836
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
                                    "value": "RunRoutine_list",
                                    "loc": {
                                      "start": 2858,
                                      "end": 2873
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 2855,
                                    "end": 2873
                                  }
                                }
                              ],
                              "loc": {
                                "start": 2837,
                                "end": 2887
                              }
                            },
                            "loc": {
                              "start": 2819,
                              "end": 2887
                            }
                          }
                        ],
                        "loc": {
                          "start": 2724,
                          "end": 2897
                        }
                      },
                      "loc": {
                        "start": 2719,
                        "end": 2897
                      }
                    }
                  ],
                  "loc": {
                    "start": 2694,
                    "end": 2903
                  }
                },
                "loc": {
                  "start": 2688,
                  "end": 2903
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 2908,
                    "end": 2916
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
                          "start": 2927,
                          "end": 2938
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2927,
                        "end": 2938
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorRunProject",
                        "loc": {
                          "start": 2947,
                          "end": 2966
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2947,
                        "end": 2966
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorRunRoutine",
                        "loc": {
                          "start": 2975,
                          "end": 2994
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2975,
                        "end": 2994
                      }
                    }
                  ],
                  "loc": {
                    "start": 2917,
                    "end": 3000
                  }
                },
                "loc": {
                  "start": 2908,
                  "end": 3000
                }
              }
            ],
            "loc": {
              "start": 2682,
              "end": 3004
            }
          },
          "loc": {
            "start": 2643,
            "end": 3004
          }
        }
      ],
      "loc": {
        "start": 2639,
        "end": 3006
      }
    },
    "loc": {
      "start": 2565,
      "end": 3006
    }
  },
  "variableValues": {},
  "path": {
    "key": "runProjectOrRunRoutine_findMany"
  }
} as const;
